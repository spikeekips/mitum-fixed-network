package isaac

import (
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
)

/*
StateJoiningHandler tries to join network safely. This is the basic
strategy,

* Keeping broadcasting INIT ballot with Voteproof

- waits the incoming INIT ballots, which should have Voteproof.
- if timed out, still broadcasts and waits.

* With (valid) incoming Ballot Voteproof

- validate it.

	- if height should be within *predictable* range

- if not valid, still broadcasts and waits.

- if Voteproof is INIT
	- if height is the next of local block, keeps broadcasts INIT ballot with Voteproof's round

	- if not,
		-> moves to sync.

- if Voteproof is ACCEPT
	- if height is not the next of local block,
		-> moves to syncing.

	- if next of local block,
		1. processes Proposal.
		1. check the result of new block of Proposal.
		1. if not,
			-> moves to sync.
		1. waits next INIT voteproof

* With consensused INIT Voteproof received,
	- if height is not the next of local block,
		-> moves to syncing.

	- if next of local block,
		-> moves to consesus.
*/
type StateJoiningHandler struct {
	*BaseStateHandler
	cr Round
}

func NewStateJoiningHandler(
	localstate *Localstate,
	proposalProcessor ProposalProcessor,
) (*StateJoiningHandler, error) {
	if lastBlock := localstate.LastBlock(); lastBlock == nil {
		return nil, xerrors.Errorf("last block is empty")
	}

	cs := &StateJoiningHandler{
		BaseStateHandler: NewBaseStateHandler(localstate, proposalProcessor, StateJoining),
	}
	cs.BaseStateHandler.Logging = logging.NewLogging(func(c zerolog.Context) zerolog.Context {
		return c.Str("module", "consensus-state-joining-handler")
	})
	cs.BaseStateHandler.timers = localtime.NewTimers([]string{TimerIDBroadcastingINITBallot}, false)

	if timer, err := cs.TimerBroadcastingINITBallot(
		func() time.Duration { return localstate.Policy().IntervalBroadcastingINITBallot() },
		cs.currentRound,
	); err != nil {
		return nil, err
	} else if err := cs.timers.SetTimer(TimerIDBroadcastingINITBallot, timer); err != nil {
		return nil, err
	}

	return cs, nil
}

func (cs *StateJoiningHandler) SetLogger(l logging.Logger) logging.Logger {
	_ = cs.Logging.SetLogger(l)
	_ = cs.timers.SetLogger(l)

	return cs.Log()
}

func (cs *StateJoiningHandler) Activate(ctx StateChangeContext) error {
	// starts to keep broadcasting INIT Ballot
	if err := cs.timers.StartTimers([]string{TimerIDBroadcastingINITBallot}, true); err != nil {
		return err
	}

	cs.Lock()
	defer cs.Unlock()

	l := loggerWithStateChangeContext(ctx, cs.Log())
	l.Debug().Msg("activated")

	return nil
}

func (cs *StateJoiningHandler) Deactivate(ctx StateChangeContext) error {
	cs.Lock()
	defer cs.Unlock()

	if err := cs.timers.Stop(); err != nil {
		return err
	}

	l := loggerWithStateChangeContext(ctx, cs.Log())
	l.Debug().Msg("deactivated")

	return nil
}

func (cs *StateJoiningHandler) currentRound() Round {
	cs.RLock()
	defer cs.RUnlock()

	return cs.cr
}

func (cs *StateJoiningHandler) setCurrentRound(round Round) {
	cs.Lock()
	defer cs.Unlock()

	cs.cr = round
}

// NewSeal only cares on INIT ballot and it's Voteproof.
func (cs *StateJoiningHandler) NewSeal(sl seal.Seal) error {
	var ballot Ballot
	var voteproof Voteproof
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
		voteproof = t.Voteproof()
	case ACCEPTBallot:
		ballot = t
		voteproof = t.Voteproof()
	}

	l := loggerWithVoteproof(voteproof, loggerWithBallot(ballot, cs.Log()))
	l.Debug().Msg("got ballot")

	if ballot.Stage() == StageINIT {
		switch voteproof.Stage() {
		case StageACCEPT:
			return cs.handleINITBallotAndACCEPTVoteproof(ballot.(INITBallot), voteproof)
		case StageINIT:
			return cs.handleINITBallotAndINITVoteproof(ballot.(INITBallot), voteproof)
		default:
			err := xerrors.Errorf("invalid Voteproof stage found")
			l.Error().Err(err).Msg("invalid voteproof found in init ballot")

			return err
		}
	} else if ballot.Stage() == StageACCEPT {
		switch voteproof.Stage() {
		case StageINIT:
			return cs.handleACCEPTBallotAndINITVoteproof(ballot.(ACCEPTBallot), voteproof)
		default:
			err := xerrors.Errorf("invalid Voteproof stage found")
			l.Error().Err(err).Msg("invalid voteproof found in accept ballot")

			return err
		}
	}

	err := xerrors.Errorf("invalid ballot stage found")
	l.Error().Err(err).Msg("invalid ballot found")

	return err
}

func (cs *StateJoiningHandler) handleProposal(proposal Proposal) error {
	l := cs.Log().With().
		Str("proposal_hash", proposal.Hash().String()).
		Int64("proposal_height", proposal.Height().Int64()).
		Uint64("proposal_round", proposal.Round().Uint64()).
		Logger()

	l.Debug().Msg("got proposal")

	return nil
}

func (cs *StateJoiningHandler) handleINITBallotAndACCEPTVoteproof(ballot INITBallot, voteproof Voteproof) error {
	l := loggerWithVoteproof(voteproof, loggerWithBallot(ballot, cs.Log()))
	l.Debug().Msg("INIT Ballot + ACCEPT Voteproof")

	lastBlock := cs.localstate.LastBlock()

	switch d := ballot.Height() - (lastBlock.Height() + 1); {
	case d > 0:
		l.Debug().
			Msgf("Ballot.Height() is higher than expected, %d + 1; moves to syncing", lastBlock.Height())

		return cs.ChangeState(StateSyncing, voteproof)
	case d == 0:
		l.Debug().Msg("same height; keep waiting CVP")

		return nil
	default:
		l.Debug().
			Msgf("Ballot.Height() is lower than expected, %d + 1; ignore it", lastBlock.Height())

		return nil
	}
}

func (cs *StateJoiningHandler) handleINITBallotAndINITVoteproof(ballot INITBallot, voteproof Voteproof) error {
	l := loggerWithVoteproof(voteproof, loggerWithBallot(ballot, cs.Log()))
	l.Debug().Msg("INIT Ballot + INIT Voteproof")

	lastBlock := cs.localstate.LastBlock()

	switch d := ballot.Height() - (lastBlock.Height() + 1); {
	case d == 0:
		if err := checkBlockWithINITVoteproof(lastBlock, voteproof); err != nil {
			l.Error().Err(err).Msg("expected height, checked voteproof with block")

			return err
		}

		if ballot.Round() > cs.currentRound() {
			l.Debug().
				Uint64("current_round", cs.currentRound().Uint64()).
				Msg("Voteproof.Round() is same or greater than currentRound; use this round")

			cs.setCurrentRound(ballot.Round())
		}

		l.Debug().Msg("same height; keep waiting CVP")

		return nil
	case d > 0:
		l.Debug().
			Msgf("ballotVoteproof.Height() is higher than expected, %d + 1; moves to syncing", lastBlock.Height())

		return cs.ChangeState(StateSyncing, voteproof)
	default:
		l.Debug().
			Msgf("ballotVoteproof.Height() is lower than expected, %d + 1; ignore it", lastBlock.Height())

		return nil
	}
}

func (cs *StateJoiningHandler) handleACCEPTBallotAndINITVoteproof(ballot ACCEPTBallot, voteproof Voteproof) error {
	l := loggerWithVoteproof(voteproof, loggerWithBallot(ballot, cs.Log()))
	l.Debug().Msg("ACCEPT Ballot + INIT Voteproof")

	lastBlock := cs.localstate.LastBlock()

	switch d := ballot.Height() - (lastBlock.Height() + 1); {
	case d == 0:
		if err := checkBlockWithINITVoteproof(lastBlock, voteproof); err != nil {
			l.Error().Err(err).Msg("expected height, checked voteproof with block")

			return err
		}

		// NOTE expected ACCEPT Ballot received, so will process Proposal of
		// INIT Voteproof and broadcast new ACCEPT Ballot.
		_ = cs.localstate.SetLastINITVoteproof(voteproof)

		block, err := cs.proposalProcessor.ProcessINIT(ballot.Proposal(), voteproof)
		if err != nil {
			l.Debug().Err(err).Msg("tried to process Proposal, but it is not yet received")
			return err
		}

		if ab, err := NewACCEPTBallotV0FromLocalstate(cs.localstate, voteproof.Round(), block); err != nil {
			cs.Log().Error().Err(err).Msg("failed to create ACCEPTBallot; will keep trying")
			return nil
		} else {
			al := loggerWithBallot(ab, l)
			cs.BroadcastSeal(ab)
			al.Debug().Msg("ACCEPTBallot was broadcasted")
		}

		return nil
	case d > 0:
		l.Debug().
			Msgf("Ballot.Height() is higher than expected, %d + 1; moves to syncing", lastBlock.Height())

		return cs.ChangeState(StateSyncing, voteproof)
	default:
		l.Debug().
			Msgf("Ballot.Height() is lower than expected, %d + 1; ignore it", lastBlock.Height())

		return nil
	}
}

// NewVoteproof receives Voteproof.
func (cs *StateJoiningHandler) NewVoteproof(voteproof Voteproof) error {
	l := loggerWithVoteproof(voteproof, cs.Log())

	l.Debug().Msg("got Voteproof")

	switch voteproof.Stage() {
	case StageACCEPT:
		// TODO ACCEPT Voteproof is next block of local, try to process
		// Voteproof.
		return nil
	case StageINIT:
		return cs.handleINITVoteproof(voteproof)
	default:
		err := xerrors.Errorf("unknown stage Voteproof received")
		l.Error().Err(err).Msg("invalid voteproof found")
		return err
	}
}

func (cs *StateJoiningHandler) handleINITVoteproof(voteproof Voteproof) error {
	l := loggerWithLocalstate(cs.localstate, loggerWithVoteproof(voteproof, cs.Log()))

	l.Debug().Msg("expected height; moves to consensus state")

	return cs.ChangeState(StateConsensus, voteproof)
}
