package isaac

import (
	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
)

/*
ConsensusStateJoiningHandler tries to join network safely. This is the basic
strategy,

* Keeping broadcasting INIT ballot with VoteProof

- waits the incoming INIT ballots, which should have VoteProof.
- if timed out, still broadcasts and waits.

* With (valid) incoming Ballot VoteProof

- validate it.

	- if height should be within *predictable* range

- if not valid, still broadcasts and waits.

- if VoteProof is INIT
	- if height is the next of local block, keeps broadcasts INIT ballot with VoteProof's round

	- if not,
		-> moves to sync.

- if VoteProof is ACCEPT
	- if height is not the next of local block,
		-> moves to syncing.

	- if next of local block,
		1. processes Proposal.
		1. check the result of new block of Proposal.
		1. if not,
			-> moves to sync.
		1. waits next INIT VP

* With consensused INIT VoteProof received,
	- if height is not the next of local block,
		-> moves to syncing.

	- if next of local block,
		-> moves to consesus.
*/
type ConsensusStateJoiningHandler struct {
	*BaseStateHandler
	proposalProcessor           ProposalProcessor
	broadcastingINITBallotTimer *localtime.CallbackTimer
	cr                          Round
}

func NewConsensusStateJoiningHandler(
	localState *LocalState,
	proposalProcessor ProposalProcessor,
) (*ConsensusStateJoiningHandler, error) {
	if lastBlock := localState.LastBlock(); lastBlock == nil {
		return nil, xerrors.Errorf("last block is empty")
	}

	cs := &ConsensusStateJoiningHandler{
		BaseStateHandler:  NewBaseStateHandler(localState, ConsensusStateJoining),
		proposalProcessor: proposalProcessor,
	}
	cs.BaseStateHandler.Logger = logging.NewLogger(func(c zerolog.Context) zerolog.Context {
		return c.Str("module", "consensus-state-joining-handler")
	})

	bt, err := localtime.NewCallbackTimer(
		"joining-broadcasting-init-ballot",
		cs.broadcastingINITBallot,
		localState.Policy().IntervalBroadcastingINITBallot(),
		nil,
	)
	if err != nil {
		return nil, err
	}
	cs.broadcastingINITBallotTimer = bt

	return cs, nil
}

func (cs *ConsensusStateJoiningHandler) SetLogger(l zerolog.Logger) *logging.Logger {
	_ = cs.Logger.SetLogger(l)
	_ = cs.broadcastingINITBallotTimer.SetLogger(l)

	return cs.Logger
}

func (cs *ConsensusStateJoiningHandler) Activate(ctx ConsensusStateChangeContext) error {
	// starts to keep broadcasting INIT Ballot
	if err := cs.startbroadcastingINITBallotTimer(); err != nil {
		return err
	}

	cs.Lock()
	defer cs.Unlock()

	l := loggerWithConsensusStateChangeContext(ctx, cs.Log())
	l.Debug().Msg("activated")

	return nil
}

func (cs *ConsensusStateJoiningHandler) Deactivate(ctx ConsensusStateChangeContext) error {
	if err := cs.stopbroadcastingINITBallotTimer(); err != nil {
		return err
	}

	cs.Lock()
	defer cs.Unlock()

	l := loggerWithConsensusStateChangeContext(ctx, cs.Log())
	l.Debug().Msg("deactivated")

	return nil
}

func (cs *ConsensusStateJoiningHandler) currentRound() Round {
	cs.RLock()
	defer cs.RUnlock()

	return cs.cr
}

func (cs *ConsensusStateJoiningHandler) setCurrentRound(round Round) {
	cs.Lock()
	defer cs.Unlock()

	cs.cr = round
}

func (cs *ConsensusStateJoiningHandler) startbroadcastingINITBallotTimer() error {
	if err := cs.stopbroadcastingINITBallotTimer(); err != nil {
		return err
	}

	cs.Lock()
	defer cs.Unlock()

	return cs.broadcastingINITBallotTimer.Start()
}

func (cs *ConsensusStateJoiningHandler) stopbroadcastingINITBallotTimer() error {
	cs.Lock()
	defer cs.Unlock()

	if err := cs.broadcastingINITBallotTimer.Stop(); err != nil && !xerrors.Is(err, util.DaemonAlreadyStoppedError) {
		return err
	}

	return nil
}

func (cs *ConsensusStateJoiningHandler) broadcastingINITBallot() (bool, error) {
	ib, err := NewINITBallotV0FromLocalState(cs.localState, cs.currentRound(), nil)
	if err != nil {
		cs.Log().Error().Err(err).Msg("failed to broadcast INIT ballot; will keep trying")
		return true, nil
	}

	cs.BroadcastSeal(ib)

	return true, nil
}

// NewSeal only cares on INIT ballot and it's VoteProof.
func (cs *ConsensusStateJoiningHandler) NewSeal(sl seal.Seal) error {
	var ballot Ballot
	var vp VoteProof
	switch t := sl.(type) {
	case Proposal:
		return cs.handleProposal(t)
	default:
		cs.Log().Debug().
			Str("seal_hint", sl.Hint().Verbose()).
			Str("seal_hash", sl.Hash().String()).
			Str("seal_signer", sl.Signer().String()).
			Msg("this type of Seal will be ignored")
		return nil
	case INITBallot:
		ballot = t
		vp = t.VoteProof()
	case ACCEPTBallot:
		ballot = t
		vp = t.VoteProof()
	}

	l := loggerWithVoteProof(vp, loggerWithBallot(ballot, cs.Log()))
	l.Debug().Msg("got ballot")

	if ballot.Stage() == StageINIT {
		switch vp.Stage() {
		case StageACCEPT:
			return cs.handleINITBallotAndACCEPTVoteProof(ballot.(INITBallot), vp)
		case StageINIT:
			return cs.handleINITBallotAndINITVoteProof(ballot.(INITBallot), vp)
		default:
			err := xerrors.Errorf("invalid VoteProof stage found")
			l.Error().Err(err).Send()

			return err
		}
	} else if ballot.Stage() == StageACCEPT {
		switch vp.Stage() {
		case StageINIT:
			return cs.handleACCEPTBallotAndINITVoteProof(ballot.(ACCEPTBallot), vp)
		default:
			err := xerrors.Errorf("invalid VoteProof stage found")
			l.Error().Err(err).Send()

			return err
		}
	}

	err := xerrors.Errorf("invalid ballot stage found")
	l.Error().Err(err).Send()

	return err
}

func (cs *ConsensusStateJoiningHandler) handleProposal(proposal Proposal) error {
	l := cs.Log().With().
		Str("proposal_hash", proposal.Hash().String()).
		Int64("proposal_height", proposal.Height().Int64()).
		Uint64("proposal_round", proposal.Round().Uint64()).
		Logger()

	l.Debug().Msg("got proposal")

	return nil
}

func (cs *ConsensusStateJoiningHandler) handleINITBallotAndACCEPTVoteProof(ballot INITBallot, vp VoteProof) error {
	l := loggerWithVoteProof(vp, loggerWithBallot(ballot, cs.Log()))
	l.Debug().Msg("INIT Ballot + ACCEPT VoteProof")

	lastBlock := cs.localState.LastBlock()

	switch d := ballot.Height() - (lastBlock.Height() + 1); {
	case d > 0:
		l.Debug().
			Msgf("Ballot.Height() is higher than expected, %d + 1; moves to syncing", lastBlock.Height())

		return cs.ChangeState(ConsensusStateSyncing, vp)
	case d == 0:
		l.Debug().Msg("same height; keep waiting CVP")

		return nil
	default:
		l.Debug().
			Msgf("Ballot.Height() is lower than expected, %d + 1; ignore it", lastBlock.Height())

		return nil
	}
}

func (cs *ConsensusStateJoiningHandler) handleINITBallotAndINITVoteProof(ballot INITBallot, vp VoteProof) error {
	l := loggerWithVoteProof(vp, loggerWithBallot(ballot, cs.Log()))
	l.Debug().Msg("INIT Ballot + INIT VoteProof")

	lastBlock := cs.localState.LastBlock()

	switch d := ballot.Height() - (lastBlock.Height() + 1); {
	case d == 0:
		if err := checkBlockWithINITVoteProof(lastBlock, vp); err != nil {
			l.Error().Err(err).Send()

			return err
		}

		if ballot.Round() > cs.currentRound() {
			l.Debug().
				Uint64("current_round", cs.currentRound().Uint64()).
				Msg("VoteProof.Round() is same or greater than currentRound; use this round")

			cs.setCurrentRound(ballot.Round())
		}

		l.Debug().Msg("same height; keep waiting CVP")

		return nil
	case d > 0:
		l.Debug().
			Msgf("ballotVoteProof.Height() is higher than expected, %d + 1; moves to syncing", lastBlock.Height())

		return cs.ChangeState(ConsensusStateSyncing, vp)
	default:
		l.Debug().
			Msgf("ballotVoteProof.Height() is lower than expected, %d + 1; ignore it", lastBlock.Height())

		return nil
	}
}

func (cs *ConsensusStateJoiningHandler) handleACCEPTBallotAndINITVoteProof(ballot ACCEPTBallot, vp VoteProof) error {
	l := loggerWithVoteProof(vp, loggerWithBallot(ballot, cs.Log()))
	l.Debug().Msg("ACCEPT Ballot + INIT VoteProof")

	lastBlock := cs.localState.LastBlock()

	switch d := ballot.Height() - (lastBlock.Height() + 1); {
	case d == 0:
		if err := checkBlockWithINITVoteProof(lastBlock, vp); err != nil {
			l.Error().Err(err).Send()

			return err
		}

		// NOTE expected ACCEPT Ballot received, so will process Proposal of
		// INIT VoteProof and broadcast new ACCEPT Ballot.
		_ = cs.localState.SetLastINITVoteProof(vp)

		newBlock, err := cs.proposalProcessor.Process(ballot.Proposal(), nil)
		if err != nil {
			l.Debug().Err(err).Msg("tried to process Proposal, but it is not yet received")
			return err
		}

		ab, err := NewACCEPTBallotV0FromLocalState(cs.localState, vp.Round(), newBlock, nil)
		if err != nil {
			cs.Log().Error().Err(err).Msg("failed to create ACCEPTBallot; will keep trying")
			return nil
		}

		al := loggerWithBallot(ab, l)
		cs.BroadcastSeal(ab)

		al.Debug().Msg("ACCEPTBallot was broadcasted")

		return nil
	case d > 0:
		l.Debug().
			Msgf("Ballot.Height() is higher than expected, %d + 1; moves to syncing", lastBlock.Height())

		return cs.ChangeState(ConsensusStateSyncing, vp)
	default:
		l.Debug().
			Msgf("Ballot.Height() is lower than expected, %d + 1; ignore it", lastBlock.Height())

		return nil
	}
}

// NewVoteProof receives VoteProof. If received, stop broadcasting INIT ballot.
func (cs *ConsensusStateJoiningHandler) NewVoteProof(vp VoteProof) error {
	if err := cs.stopbroadcastingINITBallotTimer(); err != nil {
		return err
	}

	l := loggerWithVoteProof(vp, cs.Log())

	l.Debug().Msg("got VoteProof")

	switch vp.Stage() {
	case StageACCEPT:
		return cs.handleACCEPTVoteProof(vp)
	case StageINIT:
		return cs.handleINITVoteProof(vp)
	default:
		err := xerrors.Errorf("unknown stage VoteProof received")
		l.Error().Err(err).Send()
		return err
	}
}

func (cs *ConsensusStateJoiningHandler) handleINITVoteProof(vp VoteProof) error {
	l := loggerWithLocalState(cs.localState, loggerWithVoteProof(vp, cs.Log()))

	l.Debug().Msg("expected height; moves to consensus state")

	return cs.ChangeState(ConsensusStateConsensus, vp)
}

func (cs *ConsensusStateJoiningHandler) handleACCEPTVoteProof(vp VoteProof) error {
	l := loggerWithLocalState(cs.localState, loggerWithVoteProof(vp, cs.Log()))

	lastBlock := cs.localState.LastBlock()

	l.Debug().Msg("expected height; processing Proposal")

	// NOTE if PreviousBlock does not match with local block, moves to
	// syncing.
	if err := checkBlockWithINITVoteProof(lastBlock, vp); err != nil {
		l.Error().Err(err).Send()

		return cs.ChangeState(ConsensusStateSyncing, vp)
	}

	// processing Proposal
	fact, ok := vp.Majority().(ACCEPTBallotFact)
	if !ok {
		return xerrors.Errorf("needs ACCEPTBallotFact: fact=%T", vp.Majority())
	}

	lc := loggerWithVoteProof(vp, l).With().
		Str("proposal", fact.Proposal().String()).
		Str("new_block", fact.NewBlock().String()).
		Logger()

	newBlock, err := cs.proposalProcessor.Process(fact.Proposal(), nil)
	if err != nil {
		return err
	}

	if !fact.NewBlock().Equal(newBlock.Hash()) {
		err := xerrors.Errorf(
			"processed new block does not match; fact=%s processed=%s",
			fact.NewBlock(),
			newBlock.Hash(),
		)
		lc.Error().Err(err).Send()

		return err
	}

	_ = cs.localState.SetLastACCEPTVoteProof(vp)
	_ = cs.localState.SetLastBlock(newBlock)

	lc.Info().Msg("new block stored using ACCEPT VoteProof")

	return nil
}
