package isaac

import (
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
)

/*
StateConsensusHandler joins network consensus.

What does consensus state means?

- Block states are synced with the network.
- Node can participate every vote stages.

Consensus state is started by new INIT Voteproof and waits next Proposal.
*/
type StateConsensusHandler struct {
	*BaseStateHandler
	suffrage      Suffrage
	proposalMaker *ProposalMaker
	ballotTimer   util.Daemon
}

func NewStateConsensusHandler(
	localstate *Localstate,
	proposalProcessor ProposalProcessor,
	suffrage Suffrage,
	proposalMaker *ProposalMaker,
) (*StateConsensusHandler, error) {
	if lastBlock := localstate.LastBlock(); lastBlock == nil {
		return nil, xerrors.Errorf("last block is empty")
	}

	cs := &StateConsensusHandler{
		BaseStateHandler: NewBaseStateHandler(localstate, proposalProcessor, StateConsensus),
		suffrage:         suffrage,
		proposalMaker:    proposalMaker,
	}
	cs.BaseStateHandler.Logger = logging.NewLogger(func(c zerolog.Context) zerolog.Context {
		return c.Str("module", "consensus-state-consensus-handler")
	})

	return cs, nil
}

func (cs *StateConsensusHandler) SetLogger(l zerolog.Logger) *logging.Logger {
	_ = cs.Logger.SetLogger(l)

	if cs.ballotTimer != nil {
		if logger, ok := cs.ballotTimer.(logging.SetLogger); ok {
			logger.SetLogger(l)
		}
	}

	return cs.Logger
}

func (cs *StateConsensusHandler) Activate(ctx StateChangeContext) error {
	if ctx.Voteproof() == nil {
		return xerrors.Errorf("consensus handler got empty Voteproof")
	} else if ctx.Voteproof().Stage() != StageINIT {
		return xerrors.Errorf("consensus handler starts with INIT Voteproof: %s", ctx.Voteproof().Stage())
	} else if err := ctx.Voteproof().IsValid(nil); err != nil {
		return xerrors.Errorf("consensus handler got invalid Voteproof: %w", err)
	}

	_ = cs.localstate.SetLastINITVoteproof(ctx.Voteproof())

	l := loggerWithStateChangeContext(ctx, cs.Log())

	go func() {
		if err := cs.handleINITVoteproof(ctx.Voteproof()); err != nil {
			l.Error().Err(err).Send()
		}
	}()

	l.Debug().Msg("activated")

	return nil
}

func (cs *StateConsensusHandler) Deactivate(ctx StateChangeContext) error {
	l := loggerWithStateChangeContext(ctx, cs.Log())

	if err := cs.stopBallotTimer(); err != nil {
		return err
	}

	l.Debug().Msg("deactivated")

	return nil
}

func (cs *StateConsensusHandler) startBallotTimer(timer util.Daemon) error {
	if err := cs.stopBallotTimer(); err != nil {
		return err
	}

	cs.Lock()
	defer cs.Unlock()

	if logger, ok := timer.(logging.SetLogger); ok {
		_ = logger.SetLogger(*cs.Log())
	}

	cs.ballotTimer = timer

	if err := cs.ballotTimer.Start(); err != nil {
		return err
	}

	return nil
}

func (cs *StateConsensusHandler) stopBallotTimer() error {
	cs.Lock()
	defer cs.Unlock()

	if cs.ballotTimer != nil {
		if err := cs.ballotTimer.Stop(); err != nil {
			return err
		}
		cs.ballotTimer = nil
	}

	return nil
}

func (cs *StateConsensusHandler) waitProposal(vp Voteproof) error { // nolint
	cs.Log().Debug().Msg("waiting proposal")

	if proposed, err := cs.proposal(vp); err != nil {
		return err
	} else if proposed {
		return nil
	}

	var timers []*localtime.CallbackTimer

	{ // check received Proposal; will be executed only onece.
		timer, err := localtime.NewCallbackTimer(
			"consensus-checking-proposal",
			func() (bool, error) {
				l := loggerWithVoteproof(vp, cs.Log())
				l.Debug().Msg("trying to check already received Proposal")

				// if Proposal already received, find and processing it.
				if proposal, err := cs.localstate.Storage().Proposal(vp.Height(), vp.Round()); err != nil {
					l.Error().Err(err).Msg("failed to check the received Proposal, but keep trying")
				} else {
					go func() {
						if err := cs.handleProposal(proposal); err != nil {
							l.Error().Err(err).Send()
						}
					}()
				}

				return false, nil
			},
			time.Millisecond*100,
			nil,
		)
		if err != nil {
			return err
		}

		timers = append(timers, timer)
	}

	{ // waiting timer
		var calledCount int64
		timer, err := localtime.NewCallbackTimer(
			"consensus-next-round-if-waiting-proposal-timeout",
			func() (bool, error) {
				vp := cs.localstate.LastINITVoteproof()

				l := loggerWithVoteproof(vp, cs.Log())

				round := vp.Round() + 1

				l.Debug().
					Dur("timeout", cs.localstate.Policy().TimeoutWaitingProposal()).
					Uint64("next_round", round.Uint64()).
					Msg("timeout; waiting Proposal; trying to move next round")

				ib, err := NewINITBallotV0FromLocalstate(cs.localstate, round, nil)
				if err != nil {
					l.Error().Err(err).Msg("failed to move next round; will keep trying")
					return true, nil
				}

				cs.BroadcastSeal(ib)

				return true, nil
			},
			0,
			func() time.Duration {
				defer atomic.AddInt64(&calledCount, 1)

				// NOTE at 1st time, wait timeout duration, after then, periodically
				// broadcast INIT Ballot.
				if atomic.LoadInt64(&calledCount) < 1 {
					return cs.localstate.Policy().TimeoutWaitingProposal()
				}

				return cs.localstate.Policy().IntervalBroadcastingINITBallot()
			},
		)
		if err != nil {
			return err
		}

		timers = append(timers, timer)
	}

	return cs.startBallotTimer(localtime.NewCallbackTimerset(timers))
}

func (cs *StateConsensusHandler) NewSeal(sl seal.Seal) error {
	switch t := sl.(type) {
	case Proposal:
		return cs.handleProposal(t)
	default:
		return nil
	}
}

func (cs *StateConsensusHandler) NewVoteproof(vp Voteproof) error {
	if err := cs.stopBallotTimer(); err != nil {
		return err
	}

	l := loggerWithVoteproof(vp, cs.Log())

	l.Debug().Msg("Voteproof received")

	// NOTE if drew, goes to next round.
	if vp.Result() == VoteproofDraw {
		return cs.startNextRound(vp)
	}

	switch vp.Stage() {
	case StageACCEPT:
		if err := cs.StoreNewBlockByVoteproof(vp); err != nil {
			l.Error().Err(err).Send()
		}

		return cs.keepBroadcastingINITBallotForNextBlock()
	case StageINIT:
		return cs.handleINITVoteproof(vp)
	default:
		err := xerrors.Errorf("invalid Voteproof received")

		l.Error().Err(err).Send()

		return err
	}
}

func (cs *StateConsensusHandler) handleINITVoteproof(vp Voteproof) error {
	l := loggerWithLocalstate(cs.localstate, loggerWithVoteproof(vp, cs.Log()))

	l.Debug().Msg("expected Voteproof received; will wait Proposal")

	return cs.waitProposal(vp)
}

func (cs *StateConsensusHandler) keepBroadcastingINITBallotForNextBlock() error {
	timer, err := localtime.NewCallbackTimer(
		"consensus-broadcasting-init-ballot-for-next-block",
		func() (bool, error) {
			vp := cs.localstate.LastINITVoteproof()

			l := loggerWithVoteproof(vp, cs.Log())

			l.Debug().Msg("trying to broadcast INIT Ballot for next block")

			ib, err := NewINITBallotV0FromLocalstate(cs.localstate, Round(0), nil)
			if err != nil {
				l.Error().Err(err).Msg("trying to broadcast INIT Ballot for next block; will keep trying")
				return true, nil
			}

			cs.BroadcastSeal(ib)

			return true, nil
		},
		cs.localstate.Policy().IntervalBroadcastingINITBallot(),
		nil,
	)
	if err != nil {
		return err
	}

	return cs.startBallotTimer(timer)
}

func (cs *StateConsensusHandler) handleProposal(proposal Proposal) error {
	l := loggerWithBallot(proposal, cs.Log())

	l.Debug().Msg("got proposal")

	// TODO if processing takes too long?
	block, err := cs.proposalProcessor.ProcessINIT(
		proposal.Hash(),
		cs.localstate.LastINITVoteproof(),
		nil,
	)
	if err != nil {
		return err
	}

	if err := cs.stopBallotTimer(); err != nil {
		return err
	}

	acting := cs.suffrage.Acting(proposal.Height(), proposal.Round())
	isActing := acting.Exists(cs.localstate.Node().Address())

	l.Debug().
		Object("acting_suffrag", acting).
		Bool("is_acting", isActing).
		Msgf("node is in acting suffrage? %v", isActing)

	if isActing {
		if err := cs.readyToSIGNBallot(proposal, block); err != nil {
			return err
		}
	}

	return cs.readyToACCEPTBallot(proposal, block)
}

func (cs *StateConsensusHandler) readyToSIGNBallot(proposal Proposal, newBlock Block) error {
	// NOTE not like broadcasting ACCEPT Ballot, SIGN Ballot will be broadcasted
	// withtout waiting.

	sb, err := NewSIGNBallotV0FromLocalstate(cs.localstate, proposal.Round(), newBlock, nil)
	if err != nil {
		cs.Log().Error().Err(err).Msg("failed to create SIGNBallot")
		return err
	}

	cs.BroadcastSeal(sb)

	loggerWithBallot(sb, cs.Log()).Debug().Msg("SIGNBallot was broadcasted")

	return nil
}

func (cs *StateConsensusHandler) readyToACCEPTBallot(proposal Proposal, newBlock Block) error {
	// NOTE if not in acting suffrage, broadcast ACCEPT Ballot after interval.
	var calledCount int64
	timer, err := localtime.NewCallbackTimer(
		"consensus-broadcasting-accept-ballot",
		func() (bool, error) {
			// TODO ACCEPTBallot should include the received SIGN Ballots.

			ab, err := NewACCEPTBallotV0FromLocalstate(cs.localstate, proposal.Round(), newBlock, nil)
			if err != nil {
				cs.Log().Error().Err(err).Msg("failed to create ACCEPTBallot; will keep trying")
				return true, nil
			}

			l := loggerWithBallot(ab, cs.Log())
			cs.BroadcastSeal(ab)

			l.Debug().Msg("ACCEPTBallot was broadcasted")

			return true, nil
		},
		0,
		func() time.Duration {
			defer atomic.AddInt64(&calledCount, 1)

			// NOTE at 1st time, wait timeout duration, after then, periodically
			// broadcast ACCEPT Ballot.
			if atomic.LoadInt64(&calledCount) < 1 {
				return cs.localstate.Policy().WaitBroadcastingACCEPTBallot()
			}

			return cs.localstate.Policy().IntervalBroadcastingACCEPTBallot()
		},
	)
	if err != nil {
		return err
	}

	return cs.startBallotTimer(timer)
}

func (cs *StateConsensusHandler) proposal(vp Voteproof) (bool, error) {
	l := loggerWithVoteproof(vp, cs.Log())

	l.Debug().Msg("prepare to broadcast Proposal")
	isProposer := cs.suffrage.IsProposer(vp.Height(), vp.Round(), cs.localstate.Node().Address())
	l.Debug().
		Object("acting_suffrag", cs.suffrage.Acting(vp.Height(), vp.Round())).
		Bool("is_acting", cs.suffrage.IsActing(vp.Height(), vp.Round(), cs.localstate.Node().Address())).
		Bool("is_proposer", isProposer).
		Msgf("node is proposer? %v", isProposer)

	if !isProposer {
		return false, nil
	}

	proposal, err := cs.proposalMaker.Proposal(vp.Round(), nil)
	if err != nil {
		return false, err
	}

	l.Debug().Interface("proposal", proposal).Msg("trying to broadcast Proposal")

	cs.BroadcastSeal(proposal)

	return true, nil
}

func (cs *StateConsensusHandler) startNextRound(vp Voteproof) error {
	cs.Log().Debug().Msg("trying to start next round")

	var round Round
	if vp.Stage() == StageACCEPT {
		round = 0
	} else {
		round = vp.Round() + 1
	}

	var calledCount int64
	timer, err := localtime.NewCallbackTimer(
		"consensus-next-round",
		func() (bool, error) {
			l := loggerWithVoteproof(vp, cs.Log()).With().
				Dur("timeout", cs.localstate.Policy().TimeoutWaitingProposal()).
				Uint64("next_round", round.Uint64()).
				Logger()

			ib, err := NewINITBallotV0FromLocalstate(cs.localstate, round, nil)
			if err != nil {
				l.Error().Err(err).Msg("failed to move next round; will keep trying")
				return true, nil
			}

			cs.BroadcastSeal(ib)

			return true, nil
		},
		0,
		func() time.Duration {
			defer atomic.AddInt64(&calledCount, 1)

			// NOTE at 1st time, wait timeout duration, after then, periodically
			// broadcast INIT Ballot.
			if atomic.LoadInt64(&calledCount) < 1 {
				return time.Nanosecond
			}

			return cs.localstate.Policy().IntervalBroadcastingINITBallot()
		},
	)
	if err != nil {
		return err
	}

	return cs.startBallotTimer(timer)
}
