package mitum

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"golang.org/x/xerrors"
)

type ConsensusStates struct {
	sync.RWMutex
	*logging.Logger
	localState *LocalState
	ballotbox  *Ballotbox
	states     map[ConsensusState]ConsensusStateHandler
	activated  ConsensusStateHandler
}

func NewConsensusStates(
	localState *LocalState,
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
		localState: localState,
		ballotbox:  ballotbox,
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

func (css *ConsensusStates) newVoteProof(vr VoteProof) error {
	return css.Activated().NewVoteProof(vr)
}

// NewSeal receives Seal and hand it over to handler;
// - Seal is considered it should be already checked IsValid().
// - if Seal is signed by LocalNode, it will be ignored.
func (css *ConsensusStates) NewSeal(sl seal.Seal) error {
	if css.Activated() == nil {
		return xerrors.Errorf("no activated handler")
	}

	log := css.Log().With().
		Str("handler", css.Activated().State().String()).
		Str("seal", sl.Hash().String()).
		Str("seal_hint", sl.Hint().Verbose()).
		Logger()

	if sl.Signer().Equal(css.localState.Node().Publickey()) {
		err := xerrors.Errorf("Seal is from LocalNode")
		log.Error().Err(err).Send()

		return err
	}

	go func() {
		if err := css.Activated().NewSeal(sl); err != nil {
			log.Error().
				Err(err).Msg("activated handler can not receive Seal")
		}
	}()

	if _, isBallot := sl.(Ballot); !isBallot {
		return nil
	}

	vr, err := css.ballotbox.Vote(sl.(Ballot))
	if err != nil {
		return err
	} else if !vr.IsFinished() {
		return nil
	}

	return css.newVoteProof(vr)
}
