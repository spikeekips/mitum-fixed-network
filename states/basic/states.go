package basicstates

import (
	"context"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
)

const (
	ContextValueError              util.ContextKey = "error"
	ContextValueStateSwitchContext util.ContextKey = "state_switch_context"
	ContextValueNewSealContext     util.ContextKey = "new_seal"
	ContextValueBlockSaved         util.ContextKey = "block_saved"
)

type States struct {
	sync.RWMutex
	*logging.Logging
	*util.ContextDaemon
	database       storage.Database
	policy         *isaac.LocalPolicy
	nodepool       *network.Nodepool
	suffrage       base.Suffrage
	statelock      sync.RWMutex
	state          base.State
	states         map[base.State]State
	statech        chan StateSwitchContext
	voteproofch    chan base.Voteproof
	proposalch     chan ballot.Proposal
	ballotbox      *isaac.Ballotbox
	timers         *localtime.Timers
	lvplock        sync.RWMutex
	lvp            base.Voteproof
	livp           base.Voteproof
	blockSavedHook *pm.Hooks
}

func NewStates(
	st storage.Database,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	ballotbox *isaac.Ballotbox,
	stoppedState, bootingState, joiningState, consensusState, syncingState State,
) (*States, error) {
	ss := &States{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "basic-states")
		}),
		database: st, policy: policy, nodepool: nodepool, suffrage: suffrage,
		state:       base.StateStopped,
		statech:     make(chan StateSwitchContext, 33),
		voteproofch: make(chan base.Voteproof, 33),
		proposalch:  make(chan ballot.Proposal, 33),
		ballotbox:   ballotbox,
		timers: localtime.NewTimers([]localtime.TimerID{
			TimerIDBroadcastJoingingINITBallot,
			TimerIDBroadcastINITBallot,
			TimerIDBroadcastProposal,
			TimerIDBroadcastACCEPTBallot,
			TimerIDSyncingWaitVoteproof,
			TimerIDFindProposal,
		}, false),
		blockSavedHook: pm.NewHooks("block-saved"),
	}

	states := map[base.State]State{
		base.StateStopped: stoppedState, base.StateBooting: bootingState,
		base.StateJoining: joiningState, base.StateConsensus: consensusState, base.StateSyncing: syncingState,
	}

	for b := range states {
		_ = states[b].SetStates(ss)
	}

	ss.states = states
	ss.ContextDaemon = util.NewContextDaemon("basic-states", ss.start)

	// NOTE set last voteproof from local
	if i := st.LastVoteproof(base.StageACCEPT); i != nil {
		ss.lvp = i
	}

	if i := st.LastVoteproof(base.StageINIT); i != nil {
		ss.livp = i
	}

	return ss, nil
}

func (ss *States) SetLogger(l logging.Logger) logging.Logger {
	_ = ss.Logging.SetLogger(l)
	_ = ss.timers.SetLogger(l)

	for b := range ss.states {
		st := ss.states[b]
		if i, ok := st.(logging.SetLogger); ok {
			_ = i.SetLogger(l)
		}
	}

	return ss.Logging.Log()
}

func (ss *States) Start() error {
	ch := ss.ContextDaemon.Wait(context.Background())

	return <-ch
}

func (ss *States) Stop() error {
	ss.Lock()
	defer ss.Unlock()

	if err := ss.ContextDaemon.Stop(); err != nil {
		if !xerrors.Is(err, util.DaemonAlreadyStoppedError) {
			return err
		}
	}

	if state := ss.State(); state != base.StateStopped {
		if st, found := ss.states[state]; found {
			if f, err := st.Exit(NewStateSwitchContext(state, base.StateStopped)); err != nil {
				return err
			} else if f != nil {
				if err := f(); err != nil {
					return err
				}
			}
		}
	}

	ss.setState(base.StateStopped)

	ss.Log().Debug().Msg("states stopped")

	return nil
}

func (ss *States) State() base.State {
	ss.statelock.RLock()
	defer ss.statelock.RUnlock()

	return ss.state
}

func (ss *States) SwitchState(sctx StateSwitchContext) error {
	if err := sctx.IsValid(nil); err != nil {
		return err
	}

	go func() {
		ss.statech <- sctx
	}()

	return nil
}

func (ss *States) Timers() *localtime.Timers {
	return ss.timers
}

func (ss *States) LastVoteproof() base.Voteproof {
	ss.lvplock.RLock()
	defer ss.lvplock.RUnlock()

	return ss.lvp
}

func (ss *States) LastINITVoteproof() base.Voteproof {
	ss.lvplock.RLock()
	defer ss.lvplock.RUnlock()

	return ss.livp
}

func (ss *States) SetLastVoteproof(voteproof base.Voteproof) bool {
	ss.lvplock.Lock()
	defer ss.lvplock.Unlock()

	if ss.lvp != nil {
		if base.CompareVoteproof(voteproof, ss.lvp) < 1 {
			return false
		}
	}

	ss.lvp = voteproof
	if voteproof.Stage() == base.StageINIT {
		ss.livp = voteproof
	}

	return true
}

func (ss *States) NewSeal(sl seal.Seal) error {
	l := isaac.LoggerWithSeal(sl, ss.Log())

	seal.LogEventWithSeal(sl, l.Debug(), true).Msg("seal received")

	if err := ss.processSeal(sl); err != nil {
		l.Error().Err(err).Msg("failed to process seal")

		return err
	}

	return nil
}

func (ss *States) NewVoteproof(voteproof base.Voteproof) {
	go func() {
		ss.voteproofch <- voteproof
	}()
}

func (ss *States) NewBlocks(blks []block.Block) error {
	ctx := context.WithValue(context.Background(), ContextValueBlockSaved, blks)

	return ss.blockSavedHook.Run(ctx)
}

func (ss *States) NewProposal(proposal ballot.Proposal) {
	go func() {
		ss.proposalch <- proposal
	}()
}

// BroadcastSeals broadcast seal to the known nodes,
// - suffrage nodes
// - and other nodes
// - if toLocal is true, sends to local
func (ss *States) BroadcastSeals(sl seal.Seal, toLocal bool) error {
	l := isaac.LoggerWithSeal(sl, ss.Log())

	seal.LogEventWithSeal(sl, l.Debug(), l.IsVerbose()).Bool("to_local", toLocal).Msg("broadcasting seal")

	if toLocal {
		go func() {
			if err := ss.NewSeal(sl); err != nil {
				l.Error().Err(err).Msg("failed to send ballot to local")
			}
		}()
	}

	// NOTE broadcast nodes of Nodepool, including suffrage nodes
	ss.nodepool.TraverseRemotes(func(node network.Node) bool {
		go func(node network.Node) {
			if err := node.Channel().SendSeal(context.TODO(), sl); err != nil {
				l.Error().Err(err).Hinted("target_node", node.Address()).Msg("failed to broadcast")
			}
		}(node)

		return true
	})

	return nil
}

func (ss *States) BlockSavedHook() *pm.Hooks {
	return ss.blockSavedHook
}

func (ss *States) setState(state base.State) {
	ss.statelock.Lock()
	defer ss.statelock.Unlock()

	ss.state = state
}

func (ss *States) start(ctx context.Context) error {
	ss.Log().Debug().Msg("states started")

	go ss.cleanBallotbox(ctx)
	go ss.detectStuck(ctx)

	errch := make(chan error)
	go ss.watch(ctx, errch)

	select {
	case err := <-errch:
		return err
	case <-ctx.Done():
		return <-errch
	}
}

func (ss *States) watch(ctx context.Context, errch chan<- error) {
end:
	for {
		var err error
		select {
		case <-ctx.Done():
			errch <- nil

			break end
		case sctx := <-ss.statech:
			err = sctx
		case voteproof := <-ss.voteproofch:
			err = ss.processVoteproof(voteproof)
		case proposal := <-ss.proposalch:
			err = ss.processProposal(proposal)
		}

		var sctx StateSwitchContext
		switch {
		case err == nil:
			continue
		case xerrors.Is(err, util.IgnoreError):
			continue
		case !xerrors.As(err, &sctx):
			ss.Log().Error().Err(err).Msg("something wrong")

			continue
		}

		if err := ss.processSwitchStates(sctx); err != nil {
			errch <- err

			break end
		} else if ss.State() == base.StateStopped {
			ss.Log().Debug().Msg("states stopped")

			errch <- sctx.Err()

			break end
		}
	}
}

func (ss *States) processSwitchStates(sctx StateSwitchContext) error {
	ss.Lock()
	defer ss.Unlock()

	for {
		var nsctx StateSwitchContext
		switch err := ss.switchState(sctx); {
		case err == nil:
			return nil
		case xerrors.As(err, &nsctx):
			sctx = nsctx
		case xerrors.Is(err, util.IgnoreError):
			ss.Log().Error().Err(err).Msg("problem during states switch, but ignore")

			return nil
		default:
			ss.Log().Error().Err(err).Msg("problem during states switch; moves to booting")

			if sctx.ToState() == base.StateBooting {
				return xerrors.Errorf("failed to move from booting to booting: %w", err)
			} else {
				<-time.After(time.Second * 1)
				sctx = NewStateSwitchContext(ss.state, base.StateBooting).SetError(err)
			}
		}
	}
}

func (ss *States) switchState(sctx StateSwitchContext) error {
	l := LoggerWithStateSwitchContext(sctx, ss.Log())

	e := l.Debug().Interface("voteproof", sctx.Voteproof())
	if err := sctx.Err(); err != nil {
		e.Err(err)
	}

	e.Msg("switching state")

	if err := sctx.IsValid(nil); err != nil {
		return xerrors.Errorf("invalid state switch context: %w", err)
	} else if ss.State() != sctx.FromState() {
		l.Debug().Msg("current state does not match in state switch context; ignore")

		return nil
	}

	if ss.State() == sctx.ToState() {
		if sctx.Voteproof() == nil {
			return xerrors.Errorf("same state, but empty voteproof")
		}

		l.Debug().Msg("processing voteproof into current state")
		if err := ss.processVoteproofInternal(sctx.Voteproof()); err != nil {
			l.Debug().Err(err).Msg("failed to process voteproof into current state; ignore")
		}

		return nil
	}

	var err error
	var nsctx StateSwitchContext
	switch err = ss.exitState(sctx); {
	case err == nil:
	case xerrors.As(err, &nsctx):
		if sctx.FromState() == nsctx.ToState() {
			l.Debug().Msg("exit to state returns same from state context; ignore")
		} else {
			l.Debug().Msg("exit to state returns state switch context; will switch state")

			return err
		}
	}

	switch err = ss.enterState(sctx); {
	case err == nil:
	case xerrors.As(err, &nsctx):
		if sctx.ToState() == nsctx.ToState() {
			l.Error().Err(xerrors.Errorf("enter to state returns same to state context; ignore"))

			err = nil
		}
	}

	l.Debug().Msg("state switched")

	return err
}

func (ss *States) exitState(sctx StateSwitchContext) error {
	l := LoggerWithStateSwitchContext(sctx, ss.Log())

	exitFunc := EmptySwitchFunc
	if f, err := ss.states[sctx.FromState()].Exit(sctx); err != nil {
		return err
	} else {
		if f != nil {
			exitFunc = f
		}

		l.Debug().Str("state", sctx.FromState().String()).Msg("state exited")
	}

	var nsctx StateSwitchContext
	switch err := exitFunc(); {
	case err == nil:
		return nil
	case xerrors.As(err, &nsctx):
		return nsctx
	default:
		l.Error().Err(err).Msg("something wrong after exit")

		return nil
	}
}

func (ss *States) enterState(sctx StateSwitchContext) error {
	l := LoggerWithStateSwitchContext(sctx, ss.Log())

	enterFunc := EmptySwitchFunc
	if f, err := ss.states[sctx.ToState()].Enter(sctx); err != nil {
		return err
	} else if f != nil {
		enterFunc = f
	}

	ss.setState(sctx.ToState())
	l.Debug().Str("state", sctx.ToState().String()).Msg("state entered; set current")

	var nsctx StateSwitchContext
	switch err := enterFunc(); {
	case err == nil:
		return nil
	case xerrors.As(err, &nsctx):
		return nsctx
	default:
		l.Error().Err(err).Msg("something wrong after entering")

		return nil
	}
}

func (ss *States) processVoteproof(voteproof base.Voteproof) error {
	ss.Lock()
	defer ss.Unlock()

	if lvp := ss.LastVoteproof(); lvp != nil {
		if base.CompareVoteproof(voteproof, lvp) < 1 {
			ss.Log().Debug().Msg("old or same height voteproof received")

			return nil
		}
	}

	return ss.processVoteproofInternal(voteproof)
}

func (ss *States) processVoteproofInternal(voteproof base.Voteproof) error {
	l := isaac.LoggerWithVoteproof(voteproof, ss.Log())
	l.Debug().HintedVerbose("voteproof", voteproof, true).Msg("new voteproof")

	vc := NewVoteproofChecker(ss.database, ss.suffrage, ss.nodepool, ss.LastVoteproof(), voteproof)
	_ = vc.SetLogger(ss.Log())

	err := util.NewChecker("voteproof-checker", []util.CheckerFunc{
		vc.CheckPoint,
		vc.CheckINITVoteproofWithLocalBlock,
		vc.CheckACCEPTVoteproofProposal,
	}).Check()

	var sctx StateSwitchContext
	switch {
	case err == nil:
	case xerrors.Is(err, SyncByVoteproofError):
		err = NewStateSwitchContext(ss.State(), base.StateSyncing).
			SetVoteproof(voteproof).
			SetError(err)
	case xerrors.Is(err, util.IgnoreError):
		err = nil
	case xerrors.As(err, &sctx):
	default:
		return err
	}

	if !ss.SetLastVoteproof(voteproof) {
		return util.IgnoreError.Errorf("old voteproof received")
	}

	if err == nil || sctx.ToState() == ss.State() {
		return ss.states[ss.State()].ProcessVoteproof(voteproof)
	} else {
		return err
	}
}

func (ss *States) processProposal(proposal ballot.Proposal) error {
	ss.Lock()
	defer ss.Unlock()

	l := isaac.LoggerWithBallot(proposal, ss.Log())
	pvc := isaac.NewProposalValidationChecker(
		ss.database, ss.suffrage, ss.nodepool,
		proposal,
		ss.LastINITVoteproof(),
	)
	_ = pvc.SetLogger(ss.Log())

	if err := util.NewChecker("proposal-validation-checker", []util.CheckerFunc{
		pvc.IsOlder,
		pvc.IsWaiting,
		pvc.IsProposer,
	}).Check(); err != nil {
		l.Error().Err(err).Msg("propossal validation failed")

		return util.IgnoreError.Wrap(err)
	}

	return ss.states[ss.State()].ProcessProposal(proposal)
}

func (ss *States) processSeal(sl seal.Seal) error {
	switch t := sl.(type) {
	case ballot.Proposal:
		if t.Node().Equal(ss.nodepool.Local().Address()) {
			return nil
		}

		if err := ss.validateProposal(t); err != nil {
			return err
		}

		if err := ss.checkBallotVoteproof(t); err != nil {
			return err
		}

		go ss.NewProposal(t)

		return nil
	case ballot.Ballot:
		if err := ss.validateBallot(t); err != nil {
			return err
		}

		switch i, err := ss.voteBallot(t); {
		case err != nil:
			return err
		case i == nil:
			return ss.checkBallotVoteproof(t)
		default:
			ss.NewVoteproof(i)

			return nil
		}
	default:
		return ss.saveSeal(sl)
	}
}

func (ss *States) saveSeal(sl seal.Seal) error {
	if err := ss.database.NewSeals([]seal.Seal{sl}); err != nil {
		if !xerrors.Is(err, storage.DuplicatedError) {
			return err
		}

		return err
	} else {
		return nil
	}
}

func (ss *States) validateProposal(proposal ballot.Proposal) error {
	pvc := isaac.NewProposalValidationChecker(
		ss.database, ss.suffrage, ss.nodepool,
		proposal,
		ss.LastINITVoteproof(),
	)
	_ = pvc.SetLogger(ss.Log())

	var fns []util.CheckerFunc
	if proposal.Node().Equal(ss.nodepool.Local().Address()) {
		fns = []util.CheckerFunc{
			pvc.SaveProposal,
			pvc.IsOlder,
		}
	} else {
		fns = []util.CheckerFunc{
			pvc.IsKnown,
			pvc.CheckSigning,
			pvc.SaveProposal,
			pvc.IsOlder,
		}
	}

	if err := util.NewChecker("proposal-validation-checker", fns).Check(); err != nil {
		switch {
		case xerrors.Is(err, isaac.KnownSealError):
		case xerrors.Is(err, util.IgnoreError):
		default:
			return err
		}

		return nil
	} else {
		return nil
	}
}

func (ss *States) validateBallot(blt ballot.Ballot) error {
	bc := NewBallotChecker(blt, ss.LastVoteproof())
	_ = bc.SetLogger(ss.Log())

	return util.NewChecker("ballot-validation-checker", []util.CheckerFunc{
		bc.CheckWithLastVoteproof,
	}).Check()
}

func (ss *States) voteBallot(blt ballot.Ballot) (base.Voteproof, error) {
	if !blt.Stage().CanVote() {
		return nil, nil
	}

	var voteproof base.Voteproof
	if i, err := ss.ballotbox.Vote(blt); err != nil {
		return nil, xerrors.Errorf("failed to vote: %w", err)
	} else {
		voteproof = i
	}

	if !voteproof.IsFinished() || voteproof.IsClosed() {
		return nil, nil
	}

	isaac.LoggerWithVoteproof(voteproof, isaac.LoggerWithBallot(blt, ss.Log())).Debug().Msg("voted and new voteproof")

	return voteproof, nil
}

func (ss *States) checkBallotVoteproof(blt ballot.Ballot) error {
	switch ss.State() {
	case base.StateJoining, base.StateConsensus, base.StateSyncing:
	default:
		return nil
	}

	if blt.Node().Equal(ss.nodepool.Local().Address()) {
		return nil
	}

	// NOTE incoming seal should be filtered as new
	var voteproof base.Voteproof
	switch t := blt.(type) {
	case ballot.INITBallot:
		voteproof = t.ACCEPTVoteproof()
	case ballot.Proposal:
		voteproof = t.Voteproof()
	case ballot.ACCEPTBallot:
		voteproof = t.Voteproof()
	default:
		return nil
	}

	l := isaac.LoggerWithVoteproof(voteproof, isaac.LoggerWithBallot(blt, ss.Log()))

	lvp := ss.LastVoteproof()
	// NOTE last init voteproof is nil, it means current database is empty, so
	// will follow current state
	if lvp == nil {
		go ss.NewVoteproof(voteproof)

		return nil
	}

	if base.CompareVoteproof(voteproof, lvp) < 1 {
		l.Debug().Msg("old or same height voteproof received")

		return nil
	} else {
		l.Debug().Msg("found higher voteproof found from ballot; process voteproof")

		go ss.NewVoteproof(voteproof)

		return nil
	}
}

func (ss *States) cleanBallotbox(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var height base.Height
			if vp := ss.LastVoteproof(); vp == nil {
				continue
			} else if vp.Height() < 1 {
				continue
			} else if height = vp.Height() - 3; height < 0 {
				continue
			}

			if err := ss.ballotbox.Clean(height); err != nil {
				ss.Log().Error().Err(err).Msg("something wrong to clean Ballotbox")
			}
		}
	}
}

func (ss *States) detectStuck(ctx context.Context) {
	ss.Log().Debug().Msg("detecting whether consensus is stuck")

	endure := time.Minute

	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()

	var stucked bool
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lvp := ss.LastVoteproof()
			state := ss.State()

			l := ss.Log().WithLogger(func(c logging.Context) logging.Emitter {
				e := c.Str("state", state.String()).Dur("endure", endure)

				if lvp != nil {
					e = e.Dict("last_voteproof", logging.Dict().
						Str("id", lvp.ID()).
						Hinted("height", lvp.Height()).
						Hinted("round", lvp.Round()).
						Hinted("stage", lvp.Stage()),
					)
				}

				return e
			})

			cc := NewConsensusStuckChecker(lvp, state, endure)
			err := util.NewChecker("consensus-stuck-checker", []util.CheckerFunc{
				cc.IsValidState,
				cc.IsOldLastVoteproofTime,
			}).Check()

			var changed bool
			if err != nil {
				if !stucked && !xerrors.Is(err, util.IgnoreError) {
					changed = true
					stucked = true
				}
			} else if stucked {
				changed = true
				stucked = false
			}

			if changed {
				if stucked {
					l.Error().Err(err).Msg("consensus stuck")
				} else {
					l.Info().Msg("consensus released from stuck")
				}
			}
		}
	}
}
