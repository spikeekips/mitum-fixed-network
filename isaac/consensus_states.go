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
	booting ConsensusStateHandler,
	joining ConsensusStateHandler,
	consensus ConsensusStateHandler,
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

	css.ActivateHandler(NewConsensusStateChangeContext(ConsensusStateStopped, ConsensusStateBooting, nil))

	return nil
}

func (css *ConsensusStates) Stop() error {
	css.Lock()
	defer css.Unlock()

	if err := css.FunctionDaemon.Stop(); err != nil {
		return err
	}

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
			l.Debug().Msgf("chaning state requested: %s -> %s", ctx.From(), ctx.To())

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
		lt := l.With().
			Str("target_node", n.Address().String()).
			Logger()

		if err := n.Channel().SendSeal(sl); err != nil {
			lt.Error().Err(err).Msg("failed to broadcast")

			if errChan != nil {
				errChan <- err
			}

			return true
		}

		lt.Debug().Msgf("seal broadcasted: %T", sl)

		return true
	})
}

func (css *ConsensusStates) newVoteProof(vp VoteProof) error {
	vpc := VoteProofChecker{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "consensus-states-voteproof-checker")
		}),
		lastBlock:         css.localState.LastBlock(),
		lastINITVoteProof: css.localState.LastINITVoteProof(),
		voteProof:         vp,
		css:               css,
	}
	_ = vpc.SetLogger(*css.Log())

	err := util.NewChecker("voteproof-checker", []util.CheckerFunc{
		vpc.CheckHeight,
		vpc.CheckINITVoteProof,
	}).Check()

	var ctx ConsensusStateToBeChangeError
	if xerrors.As(err, &ctx) {
		go func() {
			css.stateChan <- NewConsensusStateChangeContext(ctx.FromState, ctx.ToState, ctx.VoteProof)
		}()

		return nil
	} else if xerrors.Is(err, IgnoreVoteProofError) {
		return nil
	}

	if err != nil {
		return err
	}

	switch vp.Stage() {
	case StageACCEPT:
		_ = css.localState.SetLastACCEPTVoteProof(vp)
	case StageINIT:
		_ = css.localState.SetLastINITVoteProof(vp)
	}

	return css.ActiveHandler().NewVoteProof(vp)
}

// NewSeal receives Seal and hand it over to handler;
// - (TODO) Seal is considered it should be already checked IsValid().
// - if Seal is signed by LocalNode, it will be ignored.
func (css *ConsensusStates) NewSeal(sl seal.Seal) error {
	css.Log().Debug().Interface("seal", sl).Msgf("seal received: %T", sl)

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
	l := loggerWithBallot(proposal, css.Log())

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

	ivp := css.localState.LastINITVoteProof()
	l = loggerWithVoteProof(ivp, l)
	if proposal.Height() != ivp.Height() || proposal.Round() != ivp.Round() {
		err := xerrors.Errorf("unexpected Proposal received")

		l.Error().Err(err).Send()

		return err
	}

	return nil
}

func (css *ConsensusStates) vote(ballot Ballot) error {
	voteProof, err := css.ballotbox.Vote(ballot)
	if err != nil {
		return err
	}

	if !voteProof.IsFinished() {
		return nil
	}

	if voteProof.IsClosed() {
		return nil
	}

	return css.newVoteProof(voteProof)
}

func checkBlockWithINITVoteProof(block Block, vp VoteProof) error {
	// check vp.PreviousBlock with local block
	fact, ok := vp.Majority().(INITBallotFact)
	if !ok {
		return xerrors.Errorf("needs INITTBallotFact: fact=%T", vp.Majority())
	}

	if !fact.PreviousBlock().Equal(block.Hash()) {
		return xerrors.Errorf(
			"INIT VoteProof of ACCEPT Ballot has different PreviousBlock with local: previousRound=%s local=%s",

			fact.PreviousBlock(),
			block.Hash(),
		)
	}

	return nil
}
