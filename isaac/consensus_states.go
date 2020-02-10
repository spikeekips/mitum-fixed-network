package isaac

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

type ConsensusStates struct {
	sync.RWMutex
	*logging.Logger
	*util.FunctionDaemon
	localState    *LocalState
	ballotbox     *Ballotbox
	suffrage      Suffrage
	sealStorage   SealStorage
	states        map[ConsensusState]ConsensusStateHandler
	activeHandler ConsensusStateHandler
	stateChan     chan ConsensusStateChangeContext
	sealChan      chan seal.Seal
}

func NewConsensusStates(
	localState *LocalState,
	ballotbox *Ballotbox,
	suffrage Suffrage,
	sealStorage SealStorage,
	booting *ConsensusStateBootingHandler,
	joining *ConsensusStateJoiningHandler,
	consensus *ConsensusStateConsensusHandler,
	syncing ConsensusStateHandler,
	broken ConsensusStateHandler,
) *ConsensusStates {
	css := &ConsensusStates{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "consensus-states")
		}),
		localState:  localState,
		ballotbox:   ballotbox,
		suffrage:    suffrage,
		sealStorage: sealStorage,
		states: map[ConsensusState]ConsensusStateHandler{
			ConsensusStateBooting:   booting,
			ConsensusStateJoining:   joining,
			ConsensusStateConsensus: consensus,
			ConsensusStateSyncing:   syncing,
			ConsensusStateBroken:    broken,
		},
		stateChan: make(chan ConsensusStateChangeContext),
		sealChan:  make(chan seal.Seal),
	}
	css.FunctionDaemon = util.NewFunctionDaemon(css.start, false)

	for _, handler := range css.states {
		if handler == nil {
			// TODO do panic
			continue
		}

		handler.SetStateChan(css.stateChan)
		handler.SetSealChan(css.sealChan)
	}

	return css
}

func (css *ConsensusStates) SetLogger(l zerolog.Logger) *ConsensusStates {
	_ = css.Logger.SetLogger(l)
	_ = css.FunctionDaemon.SetLogger(l)

	return css
}

func (css *ConsensusStates) Start() error {
	css.Log().Debug().Msg("trying to start")
	defer css.Log().Debug().Msg("started")

	if err := css.FunctionDaemon.Start(); err != nil {
		return err
	}

	css.Lock()
	defer css.Unlock()

	css.ActivateHandler(NewConsensusStateChangeContext(ConsensusStateStopped, ConsensusStateBooting, nil))

	return nil
}

func (css *ConsensusStates) Stop() error {
	if err := css.FunctionDaemon.Stop(); err != nil {
		return err
	}

	css.Lock()
	defer css.Unlock()

	if css.activeHandler != nil {
		ctx := NewConsensusStateChangeContext(css.activeHandler.State(), ConsensusStateStopped, nil)
		if err := css.activeHandler.Deactivate(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (css *ConsensusStates) start(stopChan chan struct{}) error {
	stateStopChan := make(chan struct{})
	go css.startStateChan(stateStopChan)

	sealStopChan := make(chan struct{})
	go css.startSealChan(sealStopChan)

	<-stopChan
	stateStopChan <- struct{}{}
	sealStopChan <- struct{}{}

	return nil
}

func (css *ConsensusStates) startStateChan(stopChan chan struct{}) {
end:
	for {
		select {
		case <-stopChan:
			break end
		case ctx := <-css.stateChan:
			l := loggerWithConsensusStateChangeContext(ctx, css.Log())
			l.Debug().Msg("chaning state requested")

			if err := css.activateHandler(ctx); err != nil {
				l.Error().Err(err).Msg("failed to activate handler")
			}
		}
	}
}

func (css *ConsensusStates) startSealChan(stopChan chan struct{}) {
end:
	for {
		select {
		case <-stopChan:
			break end
		case sl := <-css.sealChan:
			go css.broadcastSeal(sl, nil)
		}
	}
}

// ActiveHandler returns the current activated handler.
func (css *ConsensusStates) ActivateHandler(ctx ConsensusStateChangeContext) {
	css.stateChan <- ctx
}

func (css *ConsensusStates) activateHandler(ctx ConsensusStateChangeContext) error {
	l := loggerWithConsensusStateChangeContext(ctx, css.Log())

	handler := css.ActiveHandler()
	if handler != nil && handler.State() == ctx.toState {
		return xerrors.Errorf("%s already activated", ctx.toState)
	}

	toHandler, found := css.states[ctx.toState]
	if !found {
		return xerrors.Errorf("unknown state found: %s", ctx.toState)
	} else if toHandler == nil { // TODO remove
		panic("next handler does not implemented")
	}

	css.Lock()
	defer css.Unlock()

	if handler != nil {
		if err := handler.Deactivate(ctx); err != nil {
			return err
		}
		l.Info().Str("handler", handler.State().String()).Msg("deactivated")
	}

	if err := toHandler.Activate(ctx); err != nil {
		return err
	}
	l.Info().Str("handler", toHandler.State().String()).Msg("activated")

	css.activeHandler = toHandler

	return nil
}

// ActiveHandler returns the current activated handler.
func (css *ConsensusStates) ActiveHandler() ConsensusStateHandler {
	css.RLock()
	defer css.RUnlock()

	return css.activeHandler
}

func (css *ConsensusStates) broadcastSeal(sl seal.Seal, errChan chan<- error) {
	l := loggerWithSeal(sl, css.Log())
	l.Debug().Msg("trying to broadcast")

	go func() {
		if err := css.NewSeal(sl); err != nil {
			l.Error().Err(err).Send()
		}
	}()

	css.localState.Nodes().Traverse(func(n Node) bool {
		go func(n Node) {
			lt := l.With().
				Str("target_node", n.Address().String()).
				Logger()

			if err := n.Channel().SendSeal(sl); err != nil {
				lt.Error().Err(err).Msg("failed to broadcast")

				if errChan != nil {
					errChan <- err
				}
				return
			}

			lt.Debug().Msg("broadcasted")
		}(n)

		return true
	})
}

func (css *ConsensusStates) newVoteProof(vp VoteProof) error {
	return css.ActiveHandler().NewVoteProof(vp)
}

// NewSeal receives Seal and hand it over to handler;
// - Seal is considered it should be already checked IsValid().
// - if Seal is signed by LocalNode, it will be ignored.
func (css *ConsensusStates) NewSeal(sl seal.Seal) error {
	if err := css.sealStorage.Add(sl); err != nil {
		return err
	}

	if css.ActiveHandler() == nil {
		return xerrors.Errorf("no activated handler")
	}

	l := loggerWithSeal(sl, css.Log()).With().
		Str("handler", css.ActiveHandler().State().String()).
		Logger()

	isFromLocal := sl.Signer().Equal(css.localState.Node().Publickey())

	if !isFromLocal {
		// TODO check validation for Seal
		if err := css.validateSeal(sl); err != nil {
			l.Error().Err(err).Msg("seal validation failed")

			return err
		}
	}

	go func() {
		if err := css.ActiveHandler().NewSeal(sl); err != nil {
			l.Error().
				Err(err).Msg("activated handler can not receive Seal")
		}
	}()

	b, isBallot := sl.(Ballot)
	if !isBallot {
		return nil
	}

	switch b.Stage() {
	case StageINIT, StageACCEPT:
		return css.vote(b)
	}

	return nil
}

func (css *ConsensusStates) validateSeal(sl seal.Seal) error {
	switch t := sl.(type) {
	case Proposal:
		return css.validateProposal(t)
	case Ballot:
		return css.validateBallot(t)
	}

	return nil
}

func (css *ConsensusStates) validateBallot(_ Ballot) error {
	// TODO check validation
	// - Ballot.Node() is in suffrage
	// - Ballot.Height() is equal or higher than LastINITVoteProof.
	// - Ballot.Round() is equal or higher than LastINITVoteProof.
	return nil
}

func (css *ConsensusStates) validateProposal(proposal Proposal) error {
	// TODO Proposal should be validated by ConsensusStates.

	l := loggerWithBallot(proposal, css.Log())

	// TODO check Proposer is valid proposer
	if !css.suffrage.IsProposer(proposal.Height(), proposal.Round(), proposal.Node()) {
		err := xerrors.Errorf(
			"wrong proposer; height=%d round=%d, but proposer=%v",
			proposal.Height(),
			proposal.Round(),
			proposal.Node(),
		)

		l.Error().Err(err).Msg("wrong proposer found")

		return err
	}

	return nil
}

func (css *ConsensusStates) vote(ballot Ballot) error {
	voteProof, err := css.ballotbox.Vote(ballot)
	if err != nil {
		return err
	} else if !voteProof.IsFinished() {
		return nil
	} else if voteProof.IsClosed() {
		return nil
	}

	return css.newVoteProof(voteProof)
}
