package isaac

import (
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type ConsensusStates struct {
	sync.RWMutex
	*logging.Logging
	*util.FunctionDaemon
	localstate    *Localstate
	ballotbox     *Ballotbox
	suffrage      base.Suffrage
	states        map[base.State]StateHandler
	activeHandler StateHandler
	stateChan     chan StateChangeContext
	sealChan      chan seal.Seal
}

func NewConsensusStates(
	localstate *Localstate,
	ballotbox *Ballotbox,
	suffrage base.Suffrage,
	booting *StateBootingHandler,
	joining *StateJoiningHandler,
	consensus *StateConsensusHandler,
	syncing *StateSyncingHandler,
	broken *StateBrokenHandler,
) *ConsensusStates {
	css := &ConsensusStates{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "consensus-states")
		}),
		localstate: localstate,
		ballotbox:  ballotbox,
		suffrage:   suffrage,
		states: map[base.State]StateHandler{
			base.StateBooting:   booting,
			base.StateJoining:   joining,
			base.StateConsensus: consensus,
			base.StateSyncing:   syncing,
			base.StateBroken:    broken,
		},
		stateChan: make(chan StateChangeContext),
		sealChan:  make(chan seal.Seal),
	}
	css.FunctionDaemon = util.NewFunctionDaemon(css.start, false)

	return css
}

func (css *ConsensusStates) SetLogger(l logging.Logger) logging.Logger {
	_ = css.Logging.SetLogger(l)
	_ = css.FunctionDaemon.SetLogger(l)

	for _, handler := range css.states {
		if handler == nil {
			continue
		}

		_ = handler.(logging.SetLogger).SetLogger(l)
	}

	return css.Log()
}

func (css *ConsensusStates) Start() error {
	css.Lock()
	defer css.Unlock()

	css.Log().Debug().Msg("trying to start")
	defer css.Log().Debug().Msg("started")

	for state, handler := range css.states {
		if handler == nil {
			css.Log().Warn().Hinted("state_handler", state).Msg("empty state handler found")
			continue
		}

		handler.SetStateChan(css.stateChan)
		handler.SetSealChan(css.sealChan)

		css.Log().Debug().Hinted("state_handler", state).Msg("state handler registered")
	}

	if err := css.FunctionDaemon.Start(); err != nil {
		return err
	}

	css.ActivateHandler(NewStateChangeContext(base.StateStopped, base.StateBooting, nil, nil))

	return nil
}

func (css *ConsensusStates) Stop() error {
	css.Lock()
	defer css.Unlock()

	if err := css.FunctionDaemon.Stop(); err != nil {
		return err
	}

	if css.activeHandler != nil {
		ctx := NewStateChangeContext(css.activeHandler.State(), base.StateStopped, nil, nil)
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

// ActivateHandler returns the current activated handler.
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
		l.Info().Hinted("handler", handler.State()).Msgf("deactivated: %s", handler.State())
	}

	if err := toHandler.Activate(ctx); err != nil {
		return err
	}
	l.Info().Hinted("handler", toHandler.State()).Msgf("activated: %s", toHandler.State())

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
	l.Debug().Msg("trying to broadcast seal")

	go func() {
		if err := css.NewSeal(sl); err != nil {
			l.Error().Err(err).Msg("failed to send ballot to local")
		}
	}()

	css.localstate.Nodes().Traverse(func(n Node) bool {
		go func(n Node) {
			lt := l.WithLogger(func(ctx logging.Context) logging.Emitter {
				return ctx.Hinted("target_node", n.Address())
			})

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

func (css *ConsensusStates) newVoteproof(voteproof base.Voteproof) error {
	vpc := NewVoteproofConsensusStateChecker(
		css.localstate.LastBlock(),
		css.localstate.LastINITVoteproof(),
		voteproof,
		css,
	)
	_ = vpc.SetLogger(css.Log())

	err := util.NewChecker("voteproof-validation-checker", []util.CheckerFunc{
		vpc.CheckHeight,
		vpc.CheckINITVoteproof,
	}).Check()

	var ctx *StateToBeChangeError
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
	case base.StageACCEPT:
		_ = css.localstate.SetLastACCEPTVoteproof(voteproof)
	case base.StageINIT:
		_ = css.localstate.SetLastINITVoteproof(voteproof)
	}

	return css.ActiveHandler().NewVoteproof(voteproof)
}

// NewSeal receives Seal and hand it over to handler;
func (css *ConsensusStates) NewSeal(sl seal.Seal) error {
	l := loggerWithSeal(sl, css.Log()).WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("handler", css.ActiveHandler().State())
	})

	l.Debug().Msg("seal received")

	if err := css.localstate.Storage().NewSeals([]seal.Seal{sl}); err != nil {
		return err
	}

	if css.ActiveHandler() == nil {
		return xerrors.Errorf("no activated handler")
	}

	isFromLocal := sl.Signer().Equal(css.localstate.Node().Publickey())

	if !isFromLocal {
		if err := css.validateSeal(sl); err != nil {
			l.Error().Err(err).Msg("seal validation failed")

			return err
		}
	}

	if blt, ok := sl.(ballot.Ballot); ok && blt.Stage().CanVote() {
		if err := css.vote(blt); err != nil {
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
	case ballot.Proposal:
		return css.validateProposal(t)
	case ballot.Ballot:
		return css.validateBallot(t)
	}

	return nil
}

func (css *ConsensusStates) validateBallot(_ ballot.Ballot) error {
	// BLOCK valite seal
	return nil
}

func (css *ConsensusStates) validateProposal(proposal ballot.Proposal) error {
	pvc := NewProposalValidationChecker(css.localstate, css.suffrage, proposal)

	return util.NewChecker("proposal-validation-checker", []util.CheckerFunc{
		pvc.IsKnown,
		pvc.CheckSigning,
		pvc.IsProposer,
		pvc.SaveProposal,
		pvc.IsOld,
	}).Check()
}

func (css *ConsensusStates) vote(blt ballot.Ballot) error {
	voteproof, err := css.ballotbox.Vote(blt)
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
	l.Debug().
		Hinted("height", voteproof.Height()).
		Hinted("round", voteproof.Round()).
		Hinted("stage", voteproof.Stage()).
		Msg("new voteproof")

	return css.newVoteproof(voteproof)
}

func checkBlockWithINITVoteproof(blk block.Block, voteproof base.Voteproof) error {
	// check voteproof.PreviousBlock with local block
	fact, ok := voteproof.Majority().(ballot.INITBallotFact)
	if !ok {
		return xerrors.Errorf("needs INITTBallotFact: fact=%T", voteproof.Majority())
	}

	if !fact.PreviousBlock().Equal(blk.Hash()) {
		return xerrors.Errorf(
			"INIT Voteproof of ACCEPT Ballot has different PreviousBlock with local: previousRound=%s local=%s",

			fact.PreviousBlock(),
			blk.Hash(),
		)
	}

	return nil
}
