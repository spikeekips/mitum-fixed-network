package isaac

import (
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
)

type ConsensusStates struct {
	sync.RWMutex
	*logging.Logger
	*util.FunctionDaemon
	localstate    *Localstate
	ballotbox     *Ballotbox
	suffrage      Suffrage
	states        map[State]StateHandler
	activeHandler StateHandler
	stateChan     chan StateChangeContext
	sealChan      chan seal.Seal
}

func NewConsensusStates(
	localstate *Localstate,
	ballotbox *Ballotbox,
	suffrage Suffrage,
	booting StateHandler,
	joining StateHandler,
	consensus StateHandler,
	syncing StateHandler,
	broken StateHandler,
) *ConsensusStates {
	css := &ConsensusStates{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "consensus-states")
		}),
		localstate: localstate,
		ballotbox:  ballotbox,
		suffrage:   suffrage,
		states: map[State]StateHandler{
			StateBooting:   booting,
			StateJoining:   joining,
			StateConsensus: consensus,
			StateSyncing:   syncing,
			StateBroken:    broken,
		},
		stateChan: make(chan StateChangeContext),
		sealChan:  make(chan seal.Seal),
	}
	css.FunctionDaemon = util.NewFunctionDaemon(css.start, false)

	return css
}

func (css *ConsensusStates) SetLogger(l zerolog.Logger) *logging.Logger {
	_ = css.Logger.SetLogger(l)
	_ = css.FunctionDaemon.SetLogger(l)

	for _, handler := range css.states {
		if handler == nil {
			continue
		}

		_ = handler.(logging.SetLogger).SetLogger(l)
	}

	return css.Logger
}

func (css *ConsensusStates) Start() error {
	css.Lock()
	defer css.Unlock()

	css.Log().Debug().Msg("trying to start")
	defer css.Log().Debug().Msg("started")

	for state, handler := range css.states {
		if handler == nil {
			css.Log().Warn().Str("state_handler", state.String()).Msg("empty state handler found")
			continue
		}

		handler.SetStateChan(css.stateChan)
		handler.SetSealChan(css.sealChan)

		css.Log().Debug().Str("state_handler", state.String()).Msg("state handler registered")
	}

	if err := css.FunctionDaemon.Start(); err != nil {
		return err
	}

	css.ActivateHandler(NewStateChangeContext(StateStopped, StateBooting, nil))

	return nil
}

func (css *ConsensusStates) Stop() error {
	css.Lock()
	defer css.Unlock()

	if err := css.FunctionDaemon.Stop(); err != nil {
		return err
	}

	if css.activeHandler != nil {
		ctx := NewStateChangeContext(css.activeHandler.State(), StateStopped, nil)
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
			l := loggerWithStateChangeContext(ctx, css.Log())
			l.Debug().Msgf("changing state requested: %s -> %s", ctx.From(), ctx.To())

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
func (css *ConsensusStates) ActivateHandler(ctx StateChangeContext) {
	css.stateChan <- ctx
}

func (css *ConsensusStates) activateHandler(ctx StateChangeContext) error {
	l := loggerWithStateChangeContext(ctx, css.Log())

	handler := css.ActiveHandler()
	if handler != nil && handler.State() == ctx.toState {
		return xerrors.Errorf("%s already activated", ctx.toState)
	}

	toHandler, found := css.states[ctx.toState]
	if !found {
		return xerrors.Errorf("unknown state found: %s", ctx.toState)
	} else if toHandler == nil {
		return xerrors.Errorf("state handler not registered: %s", ctx.toState)
	}

	css.Lock()
	defer css.Unlock()

	if handler != nil {
		if err := handler.Deactivate(ctx); err != nil {
			return err
		}
		l.Info().Str("handler", handler.State().String()).Msgf("deactivated: %s", handler.State())
	}

	if err := toHandler.Activate(ctx); err != nil {
		return err
	}
	l.Info().Str("handler", toHandler.State().String()).Msgf("activated: %s", toHandler.State())

	css.activeHandler = toHandler

	return nil
}

// ActiveHandler returns the current activated handler.
func (css *ConsensusStates) ActiveHandler() StateHandler {
	css.RLock()
	defer css.RUnlock()

	return css.activeHandler
}

func (css *ConsensusStates) broadcastSeal(sl seal.Seal, errChan chan<- error) {
	l := loggerWithSeal(sl, css.Log())
	l.Debug().Msg("trying to broadcast")

	go func() {
		if err := css.NewSeal(sl); err != nil {
			l.Error().Err(err).Msg("failed to send ballot to local")
		}
	}()

	css.localstate.Nodes().Traverse(func(n Node) bool {
		go func(n Node) {
			lt := l.With().
				Str("target_node", n.Address().String()).
				Logger()

			if err := n.Channel().SendSeal(sl); err != nil {
				lt.Error().Err(err).Msg("failed to broadcast")

				if errChan != nil {
					errChan <- err

					return
				}
			}

			lt.Debug().Msgf("seal broadcasted: %T", sl)
		}(n)

		return true
	})
}

func (css *ConsensusStates) newVoteproof(voteproof Voteproof) error {
	vpc := NewVoteproofConsensusStateChecker(
		css.localstate.LastBlock(),
		css.localstate.LastINITVoteproof(),
		voteproof,
		css,
	)
	_ = vpc.SetLogger(*css.Log())

	err := util.NewChecker("voteproof-validation-checker", []util.CheckerFunc{
		vpc.CheckHeight,
		vpc.CheckINITVoteproof,
	}).Check()

	var ctx StateToBeChangeError
	if xerrors.As(err, &ctx) {
		go func() {
			css.stateChan <- ctx.StateChangeContext()
		}()

		return nil
	} else if xerrors.Is(err, IgnoreVoteproofError) {
		return nil
	}

	if err != nil {
		return err
	}

	switch voteproof.Stage() {
	case StageACCEPT:
		_ = css.localstate.SetLastACCEPTVoteproof(voteproof)
	case StageINIT:
		_ = css.localstate.SetLastINITVoteproof(voteproof)
	}

	return css.ActiveHandler().NewVoteproof(voteproof)
}

// NewSeal receives Seal and hand it over to handler;
func (css *ConsensusStates) NewSeal(sl seal.Seal) error {
	css.Log().Debug().Interface("seal", sl).Msgf("seal received: %T", sl)

	if err := css.localstate.Storage().NewSeals([]seal.Seal{sl}); err != nil {
		return err
	}

	if css.ActiveHandler() == nil {
		return xerrors.Errorf("no activated handler")
	}

	l := loggerWithSeal(sl, css.Log()).With().
		Str("handler", css.ActiveHandler().State().String()).
		Logger()

	isFromLocal := sl.Signer().Equal(css.localstate.Node().Publickey())

	if !isFromLocal {
		if err := css.validateSeal(sl); err != nil {
			l.Error().Err(err).Msg("seal validation failed")

			return err
		}
	}

	if ballot, ok := sl.(Ballot); ok && ballot.Stage().CanVote() {
		if err := css.vote(ballot); err != nil {
			return err
		}
	}

	go func() {
		if err := css.ActiveHandler().NewSeal(sl); err != nil {
			l.Error().
				Err(err).Msg("activated handler can not receive Seal")
		}
	}()

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
	return nil
}

func (css *ConsensusStates) validateProposal(proposal Proposal) error {
	pvc := NewProposalValidationChecker(css.localstate, css.suffrage, proposal)

	return util.NewChecker("proposal-validation-checker", []util.CheckerFunc{
		pvc.IsKnown,
		pvc.IsProposer,
		pvc.SaveProposal,
		pvc.IsOld,
	}).Check()
}

func (css *ConsensusStates) vote(ballot Ballot) error {
	voteproof, err := css.ballotbox.Vote(ballot)
	if err != nil {
		return err
	}

	if !voteproof.IsFinished() {
		return nil
	}

	if voteproof.IsClosed() {
		return nil
	}

	l := loggerWithVoteproof(voteproof, css.Log())
	l.Debug().Msgf("new voteproof: %d-%d-%s", voteproof.Height().Int64(), voteproof.Round().Uint64(), voteproof.Stage())

	return css.newVoteproof(voteproof)
}

func checkBlockWithINITVoteproof(block Block, voteproof Voteproof) error {
	// check voteproof.PreviousBlock with local block
	fact, ok := voteproof.Majority().(INITBallotFact)
	if !ok {
		return xerrors.Errorf("needs INITTBallotFact: fact=%T", voteproof.Majority())
	}

	if !fact.PreviousBlock().Equal(block.Hash()) {
		return xerrors.Errorf(
			"INIT Voteproof of ACCEPT Ballot has different PreviousBlock with local: previousRound=%s local=%s",

			fact.PreviousBlock(),
			block.Hash(),
		)
	}

	return nil
}
