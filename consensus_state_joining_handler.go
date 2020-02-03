package mitum

import (
	"sync"
	"time"

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
	sync.RWMutex
	*logging.Logger
	localState                  *LocalState
	stateChan                   chan<- ConsensusStateChangeContext
	broadcastingINITBallotTimer *localtime.CallbackTimer
	cr                          Round
}

func NewConsensusStateJoiningHandler(
	localState *LocalState,
) (*ConsensusStateJoiningHandler, error) {
	cs := &ConsensusStateJoiningHandler{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "consensus-state-joining-handler")
		}),
		localState: localState,
	}

	bt, err := localtime.NewCallbackTimer(
		"joining-broadcasting-init-ballot",
		cs.broadcastingINITBallot,
		0,
		func() time.Duration {
			return localState.Policy().IntervalBroadcastingINITBallotInJoining()
		},
	)
	if err != nil {
		return nil, err
	}
	cs.broadcastingINITBallotTimer = bt

	return cs, nil
}

func (cs *ConsensusStateJoiningHandler) SetLogger(l zerolog.Logger) *logging.Logger {
	_ = cs.Logger.SetLogger(l)

	return cs.broadcastingINITBallotTimer.SetLogger(l)
}

func (cs *ConsensusStateJoiningHandler) State() ConsensusState {
	return ConsensusStateJoining
}

func (cs *ConsensusStateJoiningHandler) SetStateChan(stateChan chan<- ConsensusStateChangeContext) {
	cs.stateChan = stateChan
}

func (cs *ConsensusStateJoiningHandler) Activate() error {
	cs.Lock()
	defer cs.Unlock()

	// starts to keep broadcasting INIT Ballot
	if err := cs.startbroadcastingINITBallotTimer(); err != nil {
		return err
	}

	cs.Log().Debug().Msg("activated")

	return nil
}

func (cs *ConsensusStateJoiningHandler) Deactivate() error {
	cs.Lock()
	defer cs.Unlock()

	if err := cs.stopbroadcastingINITBallotTimer(); err != nil {
		return err
	}

	cs.Log().Debug().Msg("deactivated")

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
	vp := cs.localState.LastINITVoteProof()
	lastBlockHeight := cs.localState.LastBlockHeight()

	currentRound := Round(0)
	if lastBlockHeight <= vp.Height() {
		currentRound = vp.Round()
	}
	cs.cr = currentRound

	if err := cs.broadcastingINITBallotTimer.Stop(); err != nil {
		if !xerrors.Is(err, util.DaemonAlreadyStoppedError) {
			return err
		}
	}

	return cs.broadcastingINITBallotTimer.Start()
}

func (cs *ConsensusStateJoiningHandler) stopbroadcastingINITBallotTimer() error {
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

	cs.localState.Nodes().Traverse(func(n Node) bool {
		go func(n Node) {
			if err := n.Channel().SendSeal(ib); err != nil {
				cs.Log().Error().Err(err).Msg("failed to broadcast INIT ballot; will keep trying")
			}
		}(n)

		return true
	})

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

	log := cs.loggerWithVoteProof(vp, cs.loggerWithBallot(ballot, cs.Log()))
	log.Debug().Msg("got ballot")

	if ballot.Stage() == StageINIT {
		switch vp.Stage() {
		case StageACCEPT:
			return cs.handleINITBallotAndACCEPTVoteProof(ballot.(INITBallot), vp)
		case StageINIT:
			return cs.handleINITBallotAndINITVoteProof(ballot.(INITBallot), vp)
		default:
			err := xerrors.Errorf("invalid VoteProof stage found")
			log.Error().Err(err).Send()

			return err
		}
	} else if ballot.Stage() == StageACCEPT {
		switch vp.Stage() {
		case StageINIT:
			return cs.handleACCEPTBallotAndINITVoteProof(ballot.(ACCEPTBallot), vp)
		default:
			err := xerrors.Errorf("invalid VoteProof stage found")
			log.Error().Err(err).Send()

			return err
		}
	}

	err := xerrors.Errorf("invalid ballot stage found")
	log.Error().Err(err).Send()

	return err
}

func (cs *ConsensusStateJoiningHandler) handleProposal(proposal Proposal) error {
	log := cs.Log().With().
		Str("proposal_hash", proposal.Hash().String()).
		Int64("proposal_height", proposal.Height().Int64()).
		Uint64("proposal_round", proposal.Round().Uint64()).
		Logger()

	log.Debug().Msg("got proposal")

	return nil
}

func (cs *ConsensusStateJoiningHandler) changeState(newState ConsensusState, vp VoteProof) error {
	if newState == cs.State() {
		return xerrors.Errorf("can not change state to same joining state")
	}

	go func() {
		cs.stateChan <- ConsensusStateChangeContext{
			fromState: cs.State(),
			toState:   newState,
			voteProof: vp,
		}
	}()

	return nil
}

func (cs *ConsensusStateJoiningHandler) handleINITBallotAndACCEPTVoteProof(ballot INITBallot, vp VoteProof) error {
	// TODO check,
	// - ballot.Round() == 0
	// - ballot.Height() == vp.Height() + 1
	// - vp.Result() == VoteProofMajority

	log := cs.loggerWithVoteProof(vp, cs.loggerWithBallot(ballot, cs.Log()))
	log.Debug().Msg("> INIT Ballot + ACCEPT VoteProof")

	localHeight := cs.localState.LastBlockHeight()
	switch d := ballot.Height() - (localHeight + 1); {
	case d > 0:
		log.Debug().
			Msgf("Ballot.Height() is higher than expected, %d + 1; moves to syncing", localHeight)

		go func() {
			if err := cs.changeState(ConsensusStateSyncing, vp); err != nil {
				log.Error().Err(err).Send()
			}
		}()

		return nil
	case d == 0:
		log.Debug().Msg("same height; keep waiting CVP")

		return nil
	default:
		log.Debug().
			Msgf("Ballot.Height() is lower than expected, %d + 1; ignore it", localHeight)

		return nil
	}
}

func (cs *ConsensusStateJoiningHandler) handleINITBallotAndINITVoteProof(ballot INITBallot, vp VoteProof) error {
	// TODO check,
	// Ballot.Round() == VoteProof.Round() + 1
	// Ballot.Height() == VoteProof.Height()
	// VoteProof.Result == VoteProofMajority || VoteProofDraw

	log := cs.loggerWithVoteProof(vp, cs.loggerWithBallot(ballot, cs.Log()))
	log.Debug().Msg("> INIT Ballot + INIT VoteProof")

	localHeight := cs.localState.LastBlockHeight()
	switch d := ballot.Height() - (localHeight + 1); {
	case d == 0:
		if ballot.Round() > cs.currentRound() {
			log.Debug().
				Uint64("current_round", cs.currentRound().Uint64()).
				Msg("VoteProof.Round() is same or greater than currentRound; use this round")

			cs.setCurrentRound(ballot.Round())
		}

		log.Debug().Msg("same height; keep waiting CVP")

		return nil
	case d > 0:
		go func() {
			cs.stateChan <- ConsensusStateChangeContext{
				fromState: cs.State(),
				toState:   ConsensusStateSyncing,
				voteProof: vp,
			}
		}()
		log.Debug().
			Msgf("ballotVoteProof.Height() is higher than expected, %d + 1; moves to syncing", localHeight)

		return nil
	default:
		log.Debug().
			Msgf("ballotVoteProof.Height() is lower than expected, %d + 1; ignore it", localHeight)

		return nil
	}
}

func (cs *ConsensusStateJoiningHandler) handleACCEPTBallotAndINITVoteProof(ballot ACCEPTBallot, vp VoteProof) error {
	// TODO check,
	// - Ballot.Height() == VoteProof.Height()
	// - Ballot.Round() == VoteProof.Round()
	// - VoteProof.Result() == VoteProofMajority || VoteProofDraw

	log := cs.loggerWithVoteProof(vp, cs.loggerWithBallot(ballot, cs.Log()))
	log.Debug().Msg("> ACCEPT Ballot + INIT VoteProof")

	localHeight := cs.localState.LastBlockHeight()
	switch d := ballot.Height() - (localHeight + 1); {
	case d == 0:
		// TODO
		// 1. check Ballot.Proposal()
		// 1. process Ballot.Proposal()
		// 1. broadcast ACCEPT Ballot with the processing result
		return nil
	case d > 0:
		go func() {
			cs.stateChan <- ConsensusStateChangeContext{
				fromState: cs.State(),
				toState:   ConsensusStateSyncing,
				voteProof: vp,
			}
		}()
		log.Debug().
			Msgf("Ballot.Height() is higher than expected, %d + 1; moves to syncing", localHeight)

		return nil
	default:
		log.Debug().
			Msgf("Ballot.Height() is lower than expected, %d + 1; ignore it", localHeight)

		return nil
	}
}

// NewVoteProof receives VoteProof. If received, stop broadcasting INIT ballot.
func (cs *ConsensusStateJoiningHandler) NewVoteProof(vp VoteProof) error {
	if err := cs.stopbroadcastingINITBallotTimer(); err != nil {
		return err
	}

	log := cs.loggerWithVoteProof(vp, cs.Log())

	log.Debug().Msg("VoteProof received")

	switch vp.Stage() {
	case StageACCEPT:
		return cs.handleACCEPTVoteProof(vp)
	case StageINIT:
		return cs.handleINITVoteProof(vp)
	default:
		err := xerrors.Errorf("unknown stage VoteProof received")
		log.Error().Err(err).Send()
		return err
	}
}

func (cs *ConsensusStateJoiningHandler) handleINITVoteProof(vp VoteProof) error {
	log := cs.loggerWithLocalState(cs.loggerWithVoteProof(vp, cs.Log()))

	switch d := vp.Height() - (cs.localState.LastBlockHeight() + 1); {
	case d < 0:
		// TODO check previousBlock and previousRound. If not matched with local
		// blocks, it should be **argue** with other nodes.
		log.Debug().Msg("lower height; still wait")
		return nil
	case d > 0:
		log.Debug().Msg("hiehger height; moves to sync")
		return cs.changeState(ConsensusStateSyncing, vp)
	default:
		log.Debug().Msg("expected height; moves to consensus state")
		return cs.changeState(ConsensusStateConsensus, vp)
	}
}

func (cs *ConsensusStateJoiningHandler) handleACCEPTVoteProof(vp VoteProof) error {
	log := cs.loggerWithLocalState(cs.loggerWithVoteProof(vp, cs.Log()))

	switch d := vp.Height() - (cs.localState.LastBlockHeight() + 1); {
	case d < 0:
		// TODO check previousBlock and previousRound. If not matched with local
		// blocks, it should be **argue** with other nodes.
		log.Debug().Msg("lower height; still wait")
		return nil
	case d > 0:
		log.Debug().Msg("hiehger height; moves to sync")
		return cs.changeState(ConsensusStateSyncing, vp)
	default:
		log.Debug().Msg("expected height; processing Proposal")
		// TODO processing Proposal and then wait next INIT VoteProof.
		return nil
	}
}

func (cs *ConsensusStateJoiningHandler) loggerWithBallot(ballot Ballot, l *zerolog.Logger) *zerolog.Logger {
	ll := l.With().
		Str("seal_hint", ballot.Hint().Verbose()).
		Str("seal_hash", ballot.Hash().String()).
		Int64("ballot_height", ballot.Height().Int64()).
		Uint64("ballot_round", ballot.Round().Uint64()).
		Str("ballot_stage", ballot.Stage().String()).
		Logger()

	return &ll
}

func (cs *ConsensusStateJoiningHandler) loggerWithVoteProof(vp VoteProof, l *zerolog.Logger) *zerolog.Logger {
	ll := l.With().
		Int64("voteproof_height", vp.Height().Int64()).
		Uint64("voteproof_round", vp.Round().Uint64()).
		Str("voteproof_stage", vp.Stage().String()).
		Logger()

	return &ll
}

func (cs *ConsensusStateJoiningHandler) loggerWithLocalState(l *zerolog.Logger) *zerolog.Logger {
	ll := l.With().
		Int64("local_height", cs.localState.LastBlockHeight().Int64()).
		Uint64("current_round", cs.currentRound().Uint64()).
		Logger()

	return &ll
}
