package basicstates

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BaseConsensusState struct {
	*logging.Logging
	*BaseState
	database              storage.Database
	policy                *isaac.LocalPolicy
	nodepool              *network.Nodepool
	suffrage              base.Suffrage
	proposalMaker         *isaac.ProposalMaker
	pps                   *prprocessor.Processors
	broadcastACCEPTBallot func(
		valuehash.Hash,
		valuehash.Hash,
		base.Voteproof,
		time.Duration,
	) error
	broadcastNewINITBallot func(base.Voteproof) error
	prepareProposal        func(base.Height, base.Round, base.Voteproof) (base.Proposal, error)
	lib                    *util.LockedItem // last broadcasted INIT Ballot
}

func NewBaseConsensusState(
	st base.State,
	db storage.Database,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	proposalMaker *isaac.ProposalMaker,
	pps *prprocessor.Processors,
) *BaseConsensusState {
	bc := &BaseConsensusState{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", fmt.Sprintf("basic-%s-state", st))
		}),
		BaseState:     NewBaseState(st),
		database:      db,
		policy:        policy,
		nodepool:      nodepool,
		suffrage:      suffrage,
		proposalMaker: proposalMaker,
		pps:           pps,
		lib:           util.NewLockedItem(nil),
	}

	bc.broadcastACCEPTBallot = func(valuehash.Hash, valuehash.Hash, base.Voteproof, time.Duration) error {
		bc.Log().Debug().Msg("broadcasting accept ballot disabled")

		return nil
	}

	bc.broadcastNewINITBallot = func(base.Voteproof) error {
		bc.Log().Debug().Msg("broadcasting new init ballot disabled")

		return nil
	}

	bc.prepareProposal = func(base.Height, base.Round, base.Voteproof) (base.Proposal, error) {
		bc.Log().Debug().Msg("prepareProposal disabled")

		return nil, nil
	}

	return bc
}

// Enter starts consensus state with the voteproof,
// - height of last init voteproof + 1
func (st *BaseConsensusState) Enter(sctx StateSwitchContext) (func() error, error) {
	callback := EmptySwitchFunc
	if i, err := st.BaseState.Enter(sctx); err != nil {
		return nil, err
	} else if i != nil {
		callback = i
	}

	if err := st.checkVoteproof(sctx); err != nil {
		return nil, err
	}

	if st.proposalMaker == nil {
		st.Log().Debug().Msg("proposal maker is disabled")
	}

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		return st.ProcessVoteproof(sctx.Voteproof())
	}, nil
}

func (st *BaseConsensusState) Exit(sctx StateSwitchContext) (func() error, error) {
	callback := EmptySwitchFunc
	if i, err := st.BaseState.Exit(sctx); err != nil {
		return nil, err
	} else if i != nil {
		callback = i
	}

	return callback, nil
}

func (st *BaseConsensusState) ProcessVoteproof(voteproof base.Voteproof) error {
	if voteproof.Result() == base.VoteResultDraw { // NOTE moves to next round
		st.Log().Debug().Str("voteproof_id", voteproof.ID()).Msg("draw voteproof found; moves to next round")

		return st.nextRound(voteproof)
	}

	switch s := voteproof.Stage(); s {
	case base.StageINIT:
		return st.newINITVoteproof(voteproof)
	case base.StageACCEPT:
		return st.newACCEPTVoteproof(voteproof)
	default:
		return util.IgnoreError.Errorf("not supported voteproof stage, %v found", s)
	}
}

func (st *BaseConsensusState) checkVoteproof(sctx StateSwitchContext) error {
	if sctx.Voteproof() == nil {
		return errors.Errorf("consensus state not allowed to enter without voteproof")
	} else if stage := sctx.Voteproof().Stage(); stage != base.StageINIT {
		return errors.Errorf("consensus state not allowed to enter with init voteproof, not %v", stage)
	}

	l := st.Log().With().Str("voteproof_id", sctx.Voteproof().ID()).Logger()

	if lvp := st.LastINITVoteproof(); lvp == nil {
		return errors.Errorf("empty last init voteproof")
	} else if base.CompareVoteproof(sctx.Voteproof(), lvp) != 0 {
		h := sctx.Voteproof().Height()
		lh := lvp.Height()
		if h != lh+1 { // NOTE tolerate none-expected voteproof
			l.Error().Err(
				errors.Errorf("wrong height of voteproof, %v than last init voteproof, %v or %v + 1", h, lh, lh),
			).Msg("wrong incoming voteproof for consensus state, but enter the consensus state")

			return nil
		}
	}

	return nil
}

func (st *BaseConsensusState) nextRound(voteproof base.Voteproof) error {
	l := st.Log().With().Stringer("stage", voteproof.Stage()).
		Int64("height", voteproof.Height().Int64()).
		Uint64("round", voteproof.Round().Uint64()).
		Logger()
	l.Debug().Msg("starting next round")

	var baseBallot base.INITBallot
	{
		var err error
		switch s := voteproof.Stage(); s {
		case base.StageINIT:
			baseBallot, err = NextINITBallotFromINITVoteproof(
				st.database, st.nodepool.LocalNode(), voteproof, nil, st.policy.NetworkID())
		case base.StageACCEPT:
			baseBallot, err = NextINITBallotFromACCEPTVoteproof(
				st.database, st.nodepool.LocalNode(), voteproof, st.policy.NetworkID())
		}
		if err != nil {
			return err
		}
	}

	timer := localtime.NewContextTimer(TimerIDBroadcastINITBallot, 0, func(i int) (bool, error) {
		if i%5 == 0 {
			_ = signBallotWithFact(
				baseBallot,
				st.nodepool.LocalNode().Address(),
				st.nodepool.LocalNode().Privatekey(),
				st.policy.NetworkID(),
			)
		}

		if err := st.BroadcastBallot(baseBallot, i == 0); err != nil {
			l.Error().Err(err).Msg("failed to broadcast next round init ballot")
		}

		return true, nil
	}).SetInterval(func(i int) time.Duration {
		if i < 1 {
			return time.Nanosecond
		}

		return st.policy.IntervalBroadcastingINITBallot()
	})

	if err := st.Timers().SetTimer(timer); err != nil {
		return err
	}

	return st.Timers().StartTimers([]localtime.TimerID{
		TimerIDBroadcastINITBallot,
	}, true)
}

// ProcessProposal processes incoming proposal, not from local
func (st *BaseConsensusState) ProcessProposal(proposal base.Proposal) error {
	if err := st.broadcastProposal(proposal); err != nil {
		return err
	}

	started := time.Now()
	switch voteproof, newBlock, _ := st.processProposal(proposal); {
	case newBlock != nil:
		initialDelay := st.policy.WaitBroadcastingACCEPTBallot() - (time.Since(started))
		if initialDelay < 0 {
			initialDelay = time.Nanosecond
		}

		return st.broadcastACCEPTBallot(newBlock, proposal.Fact().Hash(), voteproof, initialDelay)
	default:
		return nil
	}
}

func (st *BaseConsensusState) newINITVoteproof(voteproof base.Voteproof) error {
	if err := st.Timers().StopTimers([]localtime.TimerID{TimerIDBroadcastProposal}); err != nil {
		return err
	}

	l := st.Log().With().Str("voteproof_id", voteproof.ID()).Logger()

	if err := st.handleUnknownINITVoteproof(voteproof); err != nil {
		return err
	}

	l.Debug().Msg("processing new init voteproof; propose proposal")

	actingSuffrage, err := st.suffrage.Acting(voteproof.Height(), voteproof.Round())
	if err != nil {
		l.Error().Err(err).Msg("failed to get acting suffrage")

		return err
	}

	// NOTE find proposal first
	var proposal base.Proposal
	switch i, err := st.findProposal(voteproof.Height(), voteproof.Round(), actingSuffrage.Proposer()); {
	case err != nil:
		return err
	case i != nil:
		l.Debug().Msg("proposal found in local")

		proposal = i
	}

	if proposal == nil && actingSuffrage.Proposer().Equal(st.nodepool.LocalNode().Address()) {
		i, err := st.prepareProposal(voteproof.Height(), voteproof.Round(), voteproof)
		if err != nil {
			return err
		}

		proposal = i
	}

	if proposal != nil { // NOTE if proposer, broadcast proposal
		return st.ProcessProposal(proposal)
	}

	// NOTE wait new proposal
	return st.whenProposalTimeout(voteproof, actingSuffrage.Proposer())
}

func (st *BaseConsensusState) newACCEPTVoteproof(voteproof base.Voteproof) error {
	if err := st.processACCEPTVoteproof(voteproof); err != nil {
		return err
	}

	return st.broadcastNewINITBallot(voteproof)
}

func (st *BaseConsensusState) processACCEPTVoteproof(voteproof base.Voteproof) error {
	fact, ok := voteproof.Majority().(base.ACCEPTBallotFact)
	if !ok {
		return errors.Errorf("needs ACCEPTBallotFact: fact=%T", voteproof.Majority())
	}

	l := st.Log().With().Str("voteproof_id", voteproof.ID()).Stringer("proposal_hash", fact.Proposal()).Logger()

	l.Debug().
		Dict("block", zerolog.Dict().
			Stringer("hash", fact.NewBlock()).
			Int64("height", voteproof.Height().Int64()).Uint64("round", voteproof.Round().Uint64())).
		Msg("processing accept voteproof")

	s := time.Now()

	proposal, facthash, err := st.processProposalOfACCEPTVoteproof(voteproof)
	switch {
	case err != nil:
		return errors.Wrap(err, "failed to process proposal of accept voteproof")
	case facthash == nil:
		return errors.Errorf("failed to process proposal of accept voteproof; empty facthash")
	}

	if proposal != nil {
		if err := st.broadcastProposal(proposal); err != nil {
			return err
		}
	}

	l.Debug().Msg("trying to store new block")
	var newBlock block.Block
	{
		var err error

		// NOTE no timeout to store block
		if result := <-st.pps.Save(context.Background(), facthash, voteproof); result.Err != nil {
			err = result.Err
		} else if newBlock = result.Block; newBlock == nil {
			err = errors.Errorf("failed to process Proposal; empty Block returned")
		}

		if err != nil {
			l.Error().Err(err).Msg("failed to save block from accept voteproof; moves to syncing")

			// NOTE if failed to save block, moves to syncing
			return st.NewStateSwitchContext(base.StateSyncing).
				SetVoteproof(voteproof).
				SetError(err)
		}
	}

	l.Info().Object("block", newBlock).Dur("elapsed", time.Since(s)).Msg("new block stored")

	return st.NewBlocks([]block.Block{newBlock})
}

func (st *BaseConsensusState) processProposalOfACCEPTVoteproof(
	voteproof base.Voteproof,
) (base.Proposal, valuehash.Hash, error) {
	fact, ok := voteproof.Majority().(base.ACCEPTBallotFact)
	if !ok {
		return nil, nil, errors.Errorf("needs ACCEPTBallotFact: fact=%T", voteproof.Majority())
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*14)
	defer cancel()

	var proposal base.Proposal
	if err := util.EnsureErrors(
		ctx,
		time.Millisecond*300,
		func() error {
			pr, found, err := st.database.Proposal(fact.Proposal())
			if err != nil {
				return err
			}

			if !found {
				return errors.Errorf("proposal of accept voteproof not found in local")
			}

			proposal = pr

			return nil
		},
		storage.ConnectionError,
		context.DeadlineExceeded,
	); err != nil {
		return nil, nil, errors.Wrap(err, "failed to find proposal of accept voteproof in local")
	}

	l := st.Log().With().Str("voteproof_id", voteproof.ID()).Stringer("proposal_fact", fact.Proposal()).Logger()

	// NOTE if proposal is not yet processed, process first.

	l.Debug().Msg("checking processing state of proposal")
	switch s := st.pps.CurrentState(fact.Proposal()); s {
	case prprocessor.Preparing, prprocessor.Prepared, prprocessor.Saving:
		l.Debug().Stringer("processor_state", s).Msg("processing proposal of accept voteproof")

		return proposal, fact.Proposal(), nil
	case prprocessor.Saved:
		l.Debug().Msg("already proposal of accept voteproof processed")

		return proposal, fact.Proposal(), nil
	default:
		l.Debug().Stringer("state", s).Msg("proposal of accept voteproof not yet processed, process it")
	}

	if _, _, ok := st.processProposal(proposal); ok {
		return proposal, fact.Proposal(), nil
	}

	switch s := st.pps.CurrentState(proposal.Fact().Hash()); s {
	case prprocessor.Preparing, prprocessor.Prepared, prprocessor.Saving:
		l.Debug().Msg("processing proposal of accept voteproof")

		return proposal, fact.Proposal(), nil
	case prprocessor.Saved:
		l.Debug().Msg("block saved from proposal of accept voteproof")

		return proposal, fact.Proposal(), nil
	default:
		return nil, nil, errors.Errorf("failed to process proposal of accept voteproof; %s", s)
	}
}

func (st *BaseConsensusState) findProposal(
	height base.Height,
	round base.Round,
	proposer base.Address,
) (base.Proposal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*14)
	defer cancel()

	var proposal base.Proposal
	err := util.EnsureErrors(
		ctx,
		time.Millisecond*300,
		func() error {
			pr, found, err := st.database.ProposalByPoint(height, round, proposer)
			if err != nil {
				return err
			}

			if !found {
				return nil
			}

			proposal = pr

			return nil
		},
		storage.ConnectionError,
		context.DeadlineExceeded,
	)

	return proposal, err
}

func (st *BaseConsensusState) processProposal(proposal base.Proposal) (base.Voteproof, valuehash.Hash, bool) {
	l := st.Log().With().Stringer("seal_hash", proposal.Hash()).Logger()

	l.Debug().Msg("processing proposal")

	voteproof := st.LastINITVoteproof()

	// NOTE if last init voteproof is not for proposal, voteproof of proposal
	// will be used.
	if pvp := proposal.BaseVoteproof(); pvp.Height() != voteproof.Height() || pvp.Round() != voteproof.Round() {
		voteproof = pvp
	}

	started := time.Now()

	result := <-st.pps.NewProposal(context.Background(), proposal.SignedFact(), voteproof)
	if result.Err != nil {
		if errors.Is(result.Err, util.IgnoreError) {
			return nil, nil, false
		}

		newBlock := valuehash.RandomSHA256WithPrefix(BlockPrefixFailedProcessProposal)
		l.Debug().Err(result.Err).Dur("elapsed", time.Since(started)).Stringer("new_block", newBlock).
			Msg("proposal processging failed; random block hash will be used")

		return voteproof, newBlock, false
	}
	newBlock := result.Block.Hash()

	l.Debug().Dur("elapsed", time.Since(started)).Stringer("new_block", newBlock).Msg("proposal processed")

	return voteproof, newBlock, true
}

func (st *BaseConsensusState) broadcastProposal(proposal base.Proposal) error {
	st.Log().Debug().Msg("broadcasting proposal")

	bpr := proposal

	timer := localtime.NewContextTimer(TimerIDBroadcastProposal, 0, func(i int) (bool, error) {
		if i%5 == 0 {
			_ = signBallot(
				bpr,
				st.nodepool.LocalNode().Privatekey(),
				st.policy.NetworkID(),
			)
		}

		if err := st.BroadcastBallot(bpr, false); err != nil {
			st.Log().Error().Err(err).Msg("failed to broadcast proposal")
		}

		return true, nil
	}).SetInterval(func(i int) time.Duration {
		if i < 1 {
			return time.Nanosecond
		}

		return st.policy.IntervalBroadcastingProposal()
	})

	if err := st.Timers().SetTimer(timer); err != nil {
		return err
	}

	return st.Timers().StartTimers([]localtime.TimerID{
		TimerIDBroadcastProposal,
		TimerIDBroadcastINITBallot,
		TimerIDBroadcastACCEPTBallot,
	}, true)
}

func (st *BaseConsensusState) defaultPrepareProposal(
	height base.Height,
	round base.Round,
	voteproof base.Voteproof,
) (base.Proposal, error) {
	if st.underHandover() {
		st.Log().Debug().Msg("under handover; will not make proposal")

		return nil, nil
	}

	if st.proposalMaker == nil {
		return nil, nil
	}

	l := st.Log().With().
		Str("voteproof_id", voteproof.ID()).
		Int64("height", height.Int64()).
		Uint64("round", round.Uint64()).
		Logger()

	l.Debug().Msg("local is proposer; preparing proposal")

	pr, err := st.proposalMaker.Proposal(height, round, voteproof)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*14)
	defer cancel()

	err = util.EnsureErrors(
		ctx,
		time.Millisecond*100,
		func() error {
			return st.database.NewProposal(pr)
		},
		storage.ConnectionError,
		context.DeadlineExceeded,
	)

	switch {
	case err == nil:
	case errors.Is(err, util.DuplicatedError):
		return pr, nil
	default:
		return nil, errors.Wrap(err, "failed to save proposal")
	}

	seal.LogEventSeal(pr, "proposal", l.Debug(), st.IsTraceLog()).Msg("proposal made")

	return pr, nil
}

func (st *BaseConsensusState) defaultBroadcastACCEPTBallot(
	newBlock,
	proposal valuehash.Hash,
	voteproof base.Voteproof,
	initialDelay time.Duration,
) error {
	baseBallot, err := ballot.NewACCEPT(
		ballot.NewACCEPTFact(
			voteproof.Height(),
			voteproof.Round(),
			proposal,
			newBlock,
		),
		st.nodepool.LocalNode().Address(),
		voteproof,
		st.nodepool.LocalNode().Privatekey(), st.policy.NetworkID(),
	)
	if err != nil {
		return errors.Wrap(err, "failed to re-sign accept ballot")
	}

	l := st.Log().With().Stringer("seal_hash", baseBallot.Hash()).Stringer("new_block", newBlock).Logger()

	l.Debug().Dur("initial_delay", initialDelay).Msg("start timer to broadcast accept ballot")

	timer := localtime.NewContextTimer(TimerIDBroadcastACCEPTBallot, 0, func(i int) (bool, error) {
		if i%5 == 0 {
			_ = signBallotWithFact(
				baseBallot,
				st.nodepool.LocalNode().Address(),
				st.nodepool.LocalNode().Privatekey(),
				st.policy.NetworkID(),
			)
		}

		if err := st.BroadcastBallot(baseBallot, i == 0); err != nil {
			l.Error().Err(err).Msg("failed to broadcast accept ballot")
		}

		return true, nil
	}).SetInterval(func(i int) time.Duration {
		// NOTE at 1st time, wait duration, after then, periodically
		// broadcast ACCEPT Ballot.
		if i < 1 {
			return initialDelay
		}

		return st.policy.IntervalBroadcastingACCEPTBallot()
	})

	if err := st.Timers().SetTimer(timer); err != nil {
		return err
	}

	return st.Timers().StartTimers([]localtime.TimerID{
		TimerIDBroadcastINITBallot,
		TimerIDBroadcastProposal,
		TimerIDBroadcastACCEPTBallot,
	}, true)
}

func (st *BaseConsensusState) defaultBroadcastNewINITBallot(voteproof base.Voteproof) error {
	if s := voteproof.Stage(); s != base.StageACCEPT {
		return errors.Errorf("for broadcastNewINITBallot, should be accept voteproof, not %v", s)
	}

	baseBallot, err := NextINITBallotFromACCEPTVoteproof(
		st.database, st.nodepool.LocalNode(), voteproof, st.policy.NetworkID())
	if err != nil {
		return err
	}

	var reused bool
	switch last := st.lastINITBallot(); {
	case last == nil:
	case last.Fact().Hash().Equal(baseBallot.Fact().Hash()):
		baseBallot = last
		reused = true
	}

	if !reused {
		_ = st.lib.Set(baseBallot)
	}

	l := st.Log().With().Int64("height", baseBallot.Fact().Height().Int64()).
		Uint64("round", baseBallot.Fact().Round().Uint64()).
		Logger()
	l.Debug().Msg("broadcasting new init ballot")

	timer := localtime.NewContextTimer(TimerIDBroadcastINITBallot, st.policy.IntervalBroadcastingINITBallot(),
		func(i int) (bool, error) {
			if i%5 == 0 {
				_ = signBallotWithFact(
					baseBallot,
					st.nodepool.LocalNode().Address(),
					st.nodepool.LocalNode().Privatekey(),
					st.policy.NetworkID(),
				)
			}

			if err := st.BroadcastBallot(baseBallot, i == 0); err != nil {
				l.Error().Err(err).Msg("failed to broadcast new init ballot")
			}

			return true, nil
		})

	if err := st.Timers().SetTimer(timer); err != nil {
		return err
	}

	return st.Timers().StartTimers([]localtime.TimerID{
		TimerIDBroadcastINITBallot,
		TimerIDBroadcastProposal,
		TimerIDBroadcastACCEPTBallot,
	}, true)
}

func (st *BaseConsensusState) whenProposalTimeout(voteproof base.Voteproof, proposer base.Address) error {
	if s := voteproof.Stage(); s != base.StageINIT {
		return errors.Errorf("for whenProposalTimeout, should be init voteproof, not %v", s)
	}

	l := st.Log().With().
		Int64("height", voteproof.Height().Int64()).
		Uint64("round", voteproof.Round().Uint64()).
		Logger()
	l.Debug().Msg("waiting new proposal; if timed out, will move to next round")

	timer, err := st.broadcastNextRoundINITBallot(
		voteproof, nil,
		func(i int) time.Duration {
			// NOTE at 1st time, wait timeout duration, after then, periodically
			// broadcast INIT Ballot.
			if i < 1 {
				return st.policy.TimeoutWaitingProposal()
			}

			return st.policy.IntervalBroadcastingINITBallot()
		},
	)
	if err != nil {
		return err
	}
	if err := st.Timers().SetTimer(timer); err != nil {
		return err
	}

	timer = localtime.NewContextTimer(TimerIDFindProposal, time.Second, func(int) (bool, error) {
		if i, err := st.findProposal(voteproof.Height(), voteproof.Round(), proposer); err == nil && i != nil {
			l.Debug().Msg("proposal found in local")

			go st.NewProposal(i)
		}

		return true, nil
	})
	if err := st.Timers().SetTimer(timer); err != nil {
		return err
	}

	return st.Timers().StartTimers([]localtime.TimerID{
		TimerIDBroadcastINITBallot,
		TimerIDBroadcastProposal,
		TimerIDFindProposal,
	}, true)
}

func (st *BaseConsensusState) handleUnknownINITVoteproof(voteproof base.Voteproof) error {
	vp, ok := voteproof.(base.VoteproofSet)
	if !ok || vp.ACCEPTVoteproof() == nil {
		return nil
	}

	l := st.Log().With().
		Int64("height", voteproof.Height().Int64()).
		Uint64("round", voteproof.Round().Uint64()).
		Logger()

	lvp := st.LastVoteproof()
	switch {
	case voteproof.Stage() != base.StageINIT:
		return errors.Errorf("for handleUnknownINITVoteproof, should be INIT voteproof, not %v", voteproof.Stage())
	case lvp == nil:
	case lvp.FinishedAt().After(localtime.UTCNow().Add(st.policy.TimeoutWaitingProposal() * -3)):
		l.Debug().Msg("next round voteproof too early; will wait")

		return nil
	}

	interval := func(i int) time.Duration {
		if i < 1 {
			return time.Nanosecond
		}

		return st.policy.IntervalBroadcastingINITBallot()
	}

	var timer localtime.Timer
	switch last := st.lastINITBallot(); {
	case last == nil:
		// NOTE last init ballot is empty, broadcast INIT Ballot
		l.Debug().Msg("empty last INIT ballot; will broadcast next round INIT ballot")

		i, err := st.broadcastNextRoundINITBallot(vp.Voteproof, vp.ACCEPTVoteproof(), interval)
		if err != nil {
			return err
		}

		timer = i
	default:
		fact := last.Fact()

		switch {
		case vp.Height() < fact.Height():
			return nil
		case vp.Height() == fact.Height() && vp.Round() < fact.Round():
			return nil
		}

		l.Debug().Msg("next round voteproof found; will broadcast next round INIT ballot")

		i, err := st.broadcastNextRoundINITBallot(vp.Voteproof, vp.ACCEPTVoteproof(), interval)
		if err != nil {
			return err
		}
		timer = i
	}

	if err := st.Timers().SetTimer(timer); err != nil {
		return err
	}

	return st.Timers().StartTimers([]localtime.TimerID{
		TimerIDBroadcastINITBallot,
	}, true)
}

func (st *BaseConsensusState) broadcastNextRoundINITBallot(
	voteproof, acceptVoteproof base.Voteproof,
	interval func(int) time.Duration,
) (localtime.Timer, error) {
	if s := voteproof.Stage(); s != base.StageINIT {
		return nil, errors.Errorf("for broadcast next round INITBallot, should be init voteproof, not %v", s)
	}

	l := st.Log().With().
		Int64("height", voteproof.Height().Int64()).
		Uint64("round", voteproof.Round().Uint64()).
		Bool("has_acceptvoteproof", acceptVoteproof == nil).
		Logger()
	l.Debug().Msg("will broadcast INIT ballot for next round")

	baseBallot, err := NextINITBallotFromINITVoteproof(
		st.database, st.nodepool.LocalNode(), voteproof, acceptVoteproof, st.policy.NetworkID())
	if err != nil {
		return nil, err
	}

	_ = st.lib.Set(baseBallot)

	var timer localtime.Timer
	timer = localtime.NewContextTimer(TimerIDBroadcastINITBallot, 0, func(i int) (bool, error) {
		if i%5 == 0 {
			_ = signBallotWithFact(
				baseBallot,
				st.nodepool.LocalNode().Address(),
				st.nodepool.LocalNode().Privatekey(),
				st.policy.NetworkID(),
			)
		}

		if err := st.BroadcastBallot(baseBallot, i == 0); err != nil {
			l.Error().Err(err).Msg("failed to broadcast next init ballot")
		}

		return true, nil
	})
	if interval != nil {
		timer = timer.SetInterval(interval)
	}

	return timer, nil
}

func (st *BaseConsensusState) lastINITBallot() base.INITBallot {
	i := st.lib.Value()
	if i == nil {
		return nil
	}

	last, ok := i.(base.INITBallot)
	if !ok {
		return nil
	}

	return last
}
