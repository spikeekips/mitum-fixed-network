package basicstates

import (
	"context"
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

var BlockPrefixFailedProcessProposal = []byte("processproposalfailed")

type ConsensusState struct {
	*logging.Logging
	*BaseState
	database      storage.Database
	policy        *isaac.LocalPolicy
	nodepool      *network.Nodepool
	suffrage      base.Suffrage
	proposalMaker *isaac.ProposalMaker
	pps           *prprocessor.Processors
}

func NewConsensusState(
	st storage.Database,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	proposalMaker *isaac.ProposalMaker,
	pps *prprocessor.Processors,
) *ConsensusState {
	return &ConsensusState{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "basic-consensus-state")
		}),
		BaseState:     NewBaseState(base.StateConsensus),
		database:      st,
		policy:        policy,
		nodepool:      nodepool,
		suffrage:      suffrage,
		proposalMaker: proposalMaker,
		pps:           pps,
	}
}

// Enter starts consensus state with the voteproof,
// - height of last init voteproof + 1
func (st *ConsensusState) Enter(sctx StateSwitchContext) (func() error, error) {
	callback := EmptySwitchFunc
	if i, err := st.BaseState.Enter(sctx); err != nil {
		return nil, err
	} else if i != nil {
		callback = i
	}

	if sctx.Voteproof() == nil {
		return nil, errors.Errorf("consensus state not allowed to enter without voteproof")
	} else if stage := sctx.Voteproof().Stage(); stage != base.StageINIT {
		return nil, errors.Errorf("consensus state not allowed to enter with init voteproof, not %v", stage)
	}

	l := st.Log().With().Str("voteproof_id", sctx.Voteproof().ID()).Logger()

	if lvp := st.LastINITVoteproof(); lvp == nil {
		return nil, errors.Errorf("empty last init voteproof")
	} else if base.CompareVoteproof(sctx.Voteproof(), lvp) != 0 {
		h := sctx.Voteproof().Height()
		lh := lvp.Height()
		if h != lh+1 { // NOTE tolerate none-expected voteproof
			l.Error().Err(
				errors.Errorf("wrong height of voteproof, %v than last init voteproof, %v or %v + 1", h, lh, lh),
			).Msg("wrong incoming voteproof for consensus state, but enter the consensus state")

			return nil, nil
		}
	}

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		return st.ProcessVoteproof(sctx.Voteproof())
	}, nil
}

func (st *ConsensusState) Exit(sctx StateSwitchContext) (func() error, error) {
	callback := EmptySwitchFunc
	if i, err := st.BaseState.Exit(sctx); err != nil {
		return nil, err
	} else if i != nil {
		callback = i
	}

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		return st.Timers().StopTimers([]localtime.TimerID{
			TimerIDBroadcastINITBallot,
			TimerIDBroadcastProposal,
			TimerIDFindProposal,
		})
	}, nil
}

func (st *ConsensusState) ProcessVoteproof(voteproof base.Voteproof) error {
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

// ProcessProposal processes incoming proposal, not from local
func (st *ConsensusState) ProcessProposal(proposal ballot.Proposal) error {
	if err := st.broadcastProposal(proposal); err != nil {
		return err
	}

	started := time.Now()
	if voteproof, newBlock, ok := st.processProposal(proposal); newBlock != nil {
		if st.suffrage.NumberOfActing() > 1 && ok {
			go func() {
				if err := st.broadcastSIGNBallot(proposal, newBlock); err != nil {
					st.Log().Error().Err(err).Msg("failed to broadcast sign ballot")
				}
			}()
		}

		initialDelay := st.policy.WaitBroadcastingACCEPTBallot() - (time.Since(started))
		if initialDelay < 0 {
			initialDelay = time.Nanosecond
		}

		return st.broadcastACCEPTBallot(newBlock, proposal.Hash(), voteproof, initialDelay)
	}
	return nil
}

func (st *ConsensusState) newINITVoteproof(voteproof base.Voteproof) error {
	if err := st.Timers().StopTimers([]localtime.TimerID{TimerIDBroadcastProposal}); err != nil {
		return err
	}

	l := st.Log().With().Str("voteproof_id", voteproof.ID()).Logger()

	l.Debug().Msg("processing new init voteproof; propose proposal")

	var proposal ballot.Proposal

	actingSuffrage, err := st.suffrage.Acting(voteproof.Height(), voteproof.Round())
	if err != nil {
		l.Error().Err(err).Msg("failed to get acting suffrage")

		return err
	}

	// NOTE find proposal first
	if i, err := st.findProposal(voteproof.Height(), voteproof.Round(), actingSuffrage.Proposer()); err != nil {
		return err
	} else if i != nil {
		l.Debug().Msg("proposal found in local")

		proposal = i
	}

	if proposal == nil && actingSuffrage.Proposer().Equal(st.nodepool.LocalNode().Address()) {
		if i, err := st.prepareProposal(voteproof.Height(), voteproof.Round(), voteproof); err != nil {
			return err
		} else if i != nil {
			proposal = i
		}
	}

	if proposal != nil { // NOTE if proposer, broadcast proposal
		return st.ProcessProposal(proposal)
	}

	// NOTE wait new proposal
	return st.whenProposalTimeout(voteproof, actingSuffrage.Proposer())
}

func (st *ConsensusState) newACCEPTVoteproof(voteproof base.Voteproof) error {
	if err := st.processACCEPTVoteproof(voteproof); err != nil {
		return err
	}

	return st.broadcastNewINITBallot(voteproof)
}

func (st *ConsensusState) processACCEPTVoteproof(voteproof base.Voteproof) error {
	fact, ok := voteproof.Majority().(ballot.ACCEPTFact)
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

	if proposal, err := st.processProposalOfACCEPTVoteproof(voteproof); err != nil {
		return errors.Wrap(err, "failed to process proposal of accept voteproof")
	} else if proposal != nil {
		if err := st.broadcastProposal(proposal); err != nil {
			return err
		}
	}

	l.Debug().Msg("trying to store new block")
	var newBlock block.Block
	{
		var err error

		// NOTE no timeout to store block
		if result := <-st.pps.Save(context.Background(), fact.Proposal(), voteproof); result.Err != nil {
			err = result.Err
		} else if newBlock = result.Block; newBlock == nil {
			err = errors.Errorf("failed to process Proposal; empty Block returned")
		}

		if err != nil {
			l.Error().Err(err).Msg("failed to save block from accept voteproof; moves to syncing")

			// NOTE if failed to save block, moves to syncing
			return NewStateSwitchContext(base.StateConsensus, base.StateSyncing).
				SetVoteproof(voteproof).
				SetError(err)
		}
	}

	l.Info().Object("block", newBlock).Dur("elapsed", time.Since(s)).Msg("new block stored")

	return st.NewBlocks([]block.Block{newBlock})
}

func (st *ConsensusState) processProposalOfACCEPTVoteproof(voteproof base.Voteproof) (ballot.Proposal, error) {
	fact, ok := voteproof.Majority().(ballot.ACCEPTFact)
	if !ok {
		return nil, errors.Errorf("needs ACCEPTBallotFact: fact=%T", voteproof.Majority())
	}

	l := st.Log().With().Str("voteproof_id", voteproof.ID()).Stringer("proposal_hash", fact.Proposal()).Logger()

	// NOTE if proposal is not yet processed, process first.

	l.Debug().Msg("checking processing state of proposal")
	switch s := st.pps.CurrentState(fact.Proposal()); s {
	case prprocessor.Preparing, prprocessor.Prepared, prprocessor.Saving:
		l.Debug().Msg("processing proposal of accept voteproof")

		return nil, nil
	case prprocessor.Saved:
		l.Debug().Msg("already proposal of accept voteproof processed")

		return nil, nil
	default:
		l.Debug().Stringer("state", s).Msg("proposal of accept voteproof not yet processed, process it")
	}

	var proposal ballot.Proposal
	switch i, found, err := st.database.Seal(fact.Proposal()); {
	case err != nil:
		return nil, errors.Wrap(err, "failed to find proposal of accept voteproof in local")
	case !found:
		return nil, errors.Errorf("proposal of accept voteproof not found in local")
	default:
		j, ok := i.(ballot.Proposal)
		if !ok {
			return nil, errors.Errorf("proposal of accept voteproof is not proposal, %T", i)
		}
		proposal = j
	}

	if _, _, ok := st.processProposal(proposal); ok {
		return proposal, nil
	}

	switch s := st.pps.CurrentState(proposal.Hash()); s {
	case prprocessor.Preparing, prprocessor.Prepared, prprocessor.Saving:
		l.Debug().Msg("processing proposal of accept voteproof")

		return proposal, nil
	case prprocessor.Saved:
		l.Debug().Msg("block saved from proposal of accept voteproof")

		return proposal, nil
	default:
		return proposal, errors.Errorf("failed to process proposal of accept voteproof")
	}
}

func (st *ConsensusState) findProposal(
	height base.Height,
	round base.Round,
	proposer base.Address,
) (ballot.Proposal, error) {
	switch i, found, err := st.database.Proposal(height, round, proposer); {
	case err != nil:
		return nil, err
	case !found:
		return nil, nil
	default:
		return i, nil
	}
}

func (st *ConsensusState) processProposal(proposal ballot.Proposal) (base.Voteproof, valuehash.Hash, bool) {
	l := st.Log().With().Stringer("seal_hash", proposal.Hash()).Logger()

	l.Debug().Msg("processing proposal")

	voteproof := st.LastINITVoteproof()

	// NOTE if last init voteproof is not for proposal, voteproof of proposal
	// will be used.
	if pvp := proposal.Voteproof(); pvp.Height() != voteproof.Height() || pvp.Round() != voteproof.Round() {
		voteproof = pvp
	}

	started := time.Now()

	result := <-st.pps.NewProposal(context.Background(), proposal, voteproof)
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

func (st *ConsensusState) broadcastProposal(proposal ballot.Proposal) error {
	st.Log().Debug().Msg("broadcasting proposal")

	timer := localtime.NewContextTimer(TimerIDBroadcastProposal, 0, func(int) (bool, error) {
		if err := st.BroadcastBallot(proposal, false); err != nil {
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

func (st *ConsensusState) prepareProposal(
	height base.Height,
	round base.Round,
	voteproof base.Voteproof,
) (ballot.Proposal, error) {
	l := st.Log().With().
		Str("voteproof_id", voteproof.ID()).
		Int64("height", height.Int64()).
		Uint64("round", round.Uint64()).
		Logger()

	l.Debug().Msg("local is proposer; preparing proposal")

	if i, err := st.proposalMaker.Proposal(height, round, voteproof); err != nil {
		return nil, err
	} else if err := st.database.NewProposal(i); err != nil { // NOTE save proposal
		if errors.Is(err, util.DuplicatedError) {
			return i, nil
		}

		return nil, errors.Wrap(err, "failed to save proposal")
	} else {
		seal.LogEventSeal(i, "proposal", l.Debug(), st.IsTraceLog()).Msg("proposal made")

		return i, nil
	}
}

func (st *ConsensusState) broadcastSIGNBallot(proposal ballot.Proposal, newBlock valuehash.Hash) error {
	st.Log().Debug().Msg("broadcasting sign ballot")

	if i, err := st.suffrage.Acting(proposal.Height(), proposal.Round()); err != nil {
		return err
	} else if !i.Exists(st.nodepool.LocalNode().Address()) {
		return nil
	}

	// NOTE not like broadcasting ACCEPT Ballot, SIGN Ballot will be broadcasted
	// withtout waiting.
	sb := ballot.NewSIGNV0(
		st.nodepool.LocalNode().Address(),
		proposal.Height(),
		proposal.Round(),
		proposal.Hash(),
		newBlock,
	)
	if err := sb.Sign(st.nodepool.LocalNode().Privatekey(), st.policy.NetworkID()); err != nil {
		return err
	} else if err := st.BroadcastBallot(sb, true); err != nil {
		return err
	} else {
		return nil
	}
}

func (st *ConsensusState) broadcastACCEPTBallot(
	newBlock,
	proposal valuehash.Hash,
	voteproof base.Voteproof,
	initialDelay time.Duration,
) error {
	baseBallot := ballot.NewACCEPTV0(
		st.nodepool.LocalNode().Address(),
		voteproof.Height(),
		voteproof.Round(),
		proposal,
		newBlock,
		voteproof,
	)

	if err := baseBallot.Sign(st.nodepool.LocalNode().Privatekey(), st.policy.NetworkID()); err != nil {
		return errors.Wrap(err, "failed to re-sign accept ballot")
	}

	l := st.Log().With().Stringer("seal_hash", baseBallot.Hash()).Stringer("new_block", newBlock).Logger()

	l.Debug().Dur("initial_delay", initialDelay).Msg("start timer to broadcast accept ballot")

	timer := localtime.NewContextTimer(TimerIDBroadcastACCEPTBallot, 0, func(i int) (bool, error) {
		if i%5 == 0 {
			_ = baseBallot.Sign(st.nodepool.LocalNode().Privatekey(), st.policy.NetworkID())
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
		TimerIDBroadcastACCEPTBallot,
		TimerIDBroadcastProposal,
	}, true)
}

func (st *ConsensusState) broadcastNewINITBallot(voteproof base.Voteproof) error {
	if s := voteproof.Stage(); s != base.StageACCEPT {
		return errors.Errorf("for broadcastNewINITBallot, should be accept voteproof, not %v", s)
	}

	var baseBallot ballot.INITV0
	if b, err := NextINITBallotFromACCEPTVoteproof(st.database, st.nodepool.LocalNode(), voteproof); err != nil {
		return err
	} else if err := b.Sign(st.nodepool.LocalNode().Privatekey(), st.policy.NetworkID()); err != nil {
		return errors.Wrap(err, "failed to re-sign new init ballot")
	} else {
		baseBallot = b
	}

	l := st.Log().With().Int64("height", baseBallot.Height().Int64()).
		Uint64("round", baseBallot.Round().Uint64()).
		Logger()
	l.Debug().Msg("broadcasting new init ballot")

	timer := localtime.NewContextTimer(TimerIDBroadcastINITBallot, st.policy.IntervalBroadcastingINITBallot(),
		func(i int) (bool, error) {
			if i%5 == 0 {
				_ = baseBallot.Sign(st.nodepool.LocalNode().Privatekey(), st.policy.NetworkID())
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
	}, true)
}

func (st *ConsensusState) whenProposalTimeout(voteproof base.Voteproof, proposer base.Address) error {
	if s := voteproof.Stage(); s != base.StageINIT {
		return errors.Errorf("for whenProposalTimeout, should be init voteproof, not %v", s)
	}

	l := st.Log().With().
		Int64("height", voteproof.Height().Int64()).
		Uint64("round", voteproof.Round().Uint64()).
		Logger()
	l.Debug().Msg("waiting new proposal; if timed out, will move to next round")

	var baseBallot ballot.INITV0
	if b, err := NextINITBallotFromINITVoteproof(st.database, st.nodepool.LocalNode(), voteproof); err != nil {
		return err
	} else if err := b.Sign(st.nodepool.LocalNode().Privatekey(), st.policy.NetworkID()); err != nil {
		return errors.Wrap(err, "failed to re-sign next init ballot")
	} else {
		baseBallot = b
	}

	var timer localtime.Timer

	timer = localtime.NewContextTimer(TimerIDBroadcastINITBallot, 0, func(i int) (bool, error) {
		if i%5 == 0 {
			_ = baseBallot.Sign(st.nodepool.LocalNode().Privatekey(), st.policy.NetworkID())
		}

		if err := st.BroadcastBallot(baseBallot, i == 0); err != nil {
			l.Error().Err(err).Msg("failed to broadcast next init ballot")
		}

		return true, nil
	}).SetInterval(func(i int) time.Duration {
		// NOTE at 1st time, wait timeout duration, after then, periodically
		// broadcast INIT Ballot.
		if i < 1 {
			return st.policy.TimeoutWaitingProposal()
		}

		return st.policy.IntervalBroadcastingINITBallot()
	})
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

func (st *ConsensusState) nextRound(voteproof base.Voteproof) error {
	l := st.Log().With().Stringer("stage", voteproof.Stage()).
		Int64("height", voteproof.Height().Int64()).
		Uint64("round", voteproof.Round().Uint64()).
		Logger()
	l.Debug().Msg("starting next round")

	var baseBallot ballot.INITV0
	{
		var err error
		switch s := voteproof.Stage(); s {
		case base.StageINIT:
			baseBallot, err = NextINITBallotFromINITVoteproof(st.database, st.nodepool.LocalNode(), voteproof)
		case base.StageACCEPT:
			baseBallot, err = NextINITBallotFromACCEPTVoteproof(st.database, st.nodepool.LocalNode(), voteproof)
		}
		if err != nil {
			return err
		}
	}

	if err := baseBallot.Sign(st.nodepool.LocalNode().Privatekey(), st.policy.NetworkID()); err != nil {
		return errors.Wrap(err, "failed to re-sign next round init ballot")
	}

	timer := localtime.NewContextTimer(TimerIDBroadcastINITBallot, 0, func(i int) (bool, error) {
		if i%5 == 0 {
			_ = baseBallot.Sign(st.nodepool.LocalNode().Privatekey(), st.policy.NetworkID())
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
