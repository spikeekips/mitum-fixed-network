package mitum

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/logging"
	"golang.org/x/xerrors"
)

type ConsensusStates struct {
	sync.RWMutex
	*logging.Logger
	ballotbox *Ballotbox
	states    map[ConsensusState]ConsensusStateHandler
	activated ConsensusStateHandler
}

func NewConsensusStates(
	ballotbox *Ballotbox,
	joining *ConsensusStateJoiningHandler,
	consensus ConsensusStateHandler,
	syncing ConsensusStateHandler,
	broken ConsensusStateHandler,
) *ConsensusStates {
	return &ConsensusStates{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "consensus-states")
		}),
		ballotbox: ballotbox,
		states: map[ConsensusState]ConsensusStateHandler{
			ConsensusStateJoining:   joining,
			ConsensusStateConsensus: consensus,
			ConsensusStateSyncing:   syncing,
			ConsensusStateBroken:    broken,
		},
	}
}

// Activated returns the current activated handler.
func (css *ConsensusStates) Activated() ConsensusStateHandler {
	css.RLock()
	defer css.RUnlock()

	return css.activated
}

// Activate activates the handler of the given ConsensusState.
func (css *ConsensusStates) Activate(cs ConsensusState) error {
	if err := cs.IsValid(nil); err != nil {
		return err
	}

	css.Lock()
	defer css.Unlock()

	if css.activated != nil {
		go func() {
			if err := css.activated.Deactivate(); err != nil {
				css.Log().Error().Err(err).
					Str("state", css.activated.State().String()).
					Msg("failed to Deactivate handler")
			}
		}()
	}

	handler := css.states[cs]
	if err := handler.Activate(); err != nil {
		return err
	}

	css.activated = handler

	return nil
}

func (css *ConsensusStates) newProposal(pr Proposal) error {
	return css.Activated().NewProposal(pr)
}

func (css *ConsensusStates) newVoteResult(vr VoteResult) error {
	return css.Activated().NewVoteResult(vr)
}

func (css *ConsensusStates) NewBallot(ballot Ballot) error {
	if css.Activated() == nil {
		return xerrors.Errorf("no activated handler")
	}

	if ballot.Stage() == StageProposal {
		return css.newProposal(ballot.(Proposal))
	}

	vr, err := css.ballotbox.Vote(ballot)
	if err != nil {
		return err
	} else if !vr.IsFinished() {
		return nil
	}

	return css.newVoteResult(vr)
}
