package basicstates

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery/memberlist"
	"github.com/spikeekips/mitum/states"
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
	database           storage.Database
	policy             *isaac.LocalPolicy
	nodepool           *network.Nodepool
	suffrage           base.Suffrage
	statelock          sync.RWMutex
	state              base.State
	states             map[base.State]State
	statech            chan StateSwitchContext
	voteproofch        chan base.Voteproof
	proposalch         chan base.Proposal
	ballotbox          *isaac.Ballotbox
	timers             *localtime.Timers
	lvplock            sync.RWMutex
	lvp                base.Voteproof
	livp               base.Voteproof
	blockSavedHook     *pm.Hooks
	isNoneSuffrageNode bool
	hd                 *Handover
	dis                *states.DiscoveryJoiner
	joinDiscoveryFunc  func(int, chan error) error
}

func NewStates( // revive:disable-line:argument-limit
	db storage.Database,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	ballotbox *isaac.Ballotbox,
	stoppedState, bootingState, joiningState, consensusState, syncingState, handoverState State,
	dis *states.DiscoveryJoiner,
	hd *Handover,
) (*States, error) {
	ss := &States{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "basic-states")
		}),
		database:    db,
		policy:      policy,
		nodepool:    nodepool,
		suffrage:    suffrage,
		state:       base.StateStopped,
		statech:     make(chan StateSwitchContext, 33),
		voteproofch: make(chan base.Voteproof, 33),
		proposalch:  make(chan base.Proposal, 33),
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
		dis:            dis,
		hd:             hd,
	}

	sts := map[base.State]State{
		base.StateStopped:   stoppedState,
		base.StateBooting:   bootingState,
		base.StateJoining:   joiningState,
		base.StateConsensus: consensusState,
		base.StateSyncing:   syncingState,
		base.StateHandover:  handoverState,
	}

	for b := range sts {
		_ = sts[b].SetStates(ss)
	}

	ss.states = sts
	ss.ContextDaemon = util.NewContextDaemon("basic-states", ss.start)

	// NOTE set last voteproof from local
	if i := db.LastVoteproof(base.StageACCEPT); i != nil {
		ss.lvp = i
	}

	if i := db.LastVoteproof(base.StageINIT); i != nil {
		ss.livp = i
	}

	ss.isNoneSuffrageNode = !suffrage.IsInside(nodepool.LocalNode().Address())

	return ss, nil
}

func (ss *States) SetLogging(l *logging.Logging) *logging.Logging {
	_ = ss.timers.SetLogging(l)

	for b := range ss.states {
		st := ss.states[b]
		if i, ok := st.(logging.SetLogging); ok {
			_ = i.SetLogging(l)
		}
	}

	return ss.Logging.SetLogging(l)
}

func (ss *States) Start() error {
	return <-ss.ContextDaemon.Wait(context.Background())
}

func (ss *States) Stop() error {
	ss.Lock()
	defer ss.Unlock()

	if err := ss.ContextDaemon.Stop(); err != nil {
		if !errors.Is(err, util.DaemonAlreadyStoppedError) {
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

	ss.Log().Debug().Str("voteproof_id", voteproof.ID()).Msg("last voteproof updated")

	return true
}

func (ss *States) NewSeal(sl seal.Seal) error {
	l := ss.Log().With().Stringer("seal_hash", sl.Hash()).Logger()

	l.Debug().Dict("seal", LogSeal(sl)).Msg("seal received")
	l.Trace().Interface("seal", sl).Msg("seal received")

	var err error
	switch t := sl.(type) {
	case base.Proposal:
		err = ss.newSealProposal(t)
	case base.Ballot:
		err = ss.newSealBallot(t)
	case operation.Seal:
		err = ss.newOperationSeal(t)
	}

	if err != nil {
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

func (ss *States) NewProposal(proposal base.Proposal) {
	go func() {
		ss.proposalch <- proposal
	}()
}

// BroadcastBallot broadcast seal to the known nodes,
// - suffrage nodes
// - if toLocal is true, sends to local
func (ss *States) BroadcastBallot(blt base.Ballot, toLocal bool) {
	go ss.broadcast(blt, toLocal, func(n base.Node) bool {
		return ss.suffrage.IsInside(n.Address())
	})
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
	if ss.joinDiscoveryFunc == nil {
		ss.joinDiscoveryFunc = ss.defaultJoinDiscovery
	}

	ss.Log().Debug().Bool("is_none_suffrage", ss.isNoneSuffrageNode).Msg("states started")

	if ss.ballotbox != nil {
		go ss.cleanBallotbox(ctx)
	}

	if err := ss.join(); err != nil {
		return err
	}

	if !ss.isNoneSuffrageNode {
		go ss.detectStuck(ctx)
	}

	errch := make(chan error)
	go ss.watch(ctx, errch)

	return <-errch
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
		case errors.Is(err, util.IgnoreError):
			continue
		case !errors.As(err, &sctx):
			ss.Log().Error().Err(err).Str("details", fmt.Sprintf("%s+v", err)).Msg("something wrong")

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
	for {
		var nsctx StateSwitchContext
		switch err := ss.switchState(sctx); {
		case err == nil:
			return nil
		case errors.As(err, &nsctx):
			sctx = nsctx
		case errors.Is(err, util.IgnoreError):
			ss.Log().Error().Err(err).Msg("problem during states switch, but ignore")

			return nil
		default:
			ss.Log().Error().Err(err).Msg("problem during states switch; moves to booting")

			if sctx.ToState() == base.StateBooting {
				return errors.Wrap(err, "failed to move to booting")
			}
			<-time.After(time.Second * 1)
			sctx = NewStateSwitchContext(ss.state, base.StateBooting).SetError(err)
		}
	}
}

func (ss *States) switchState(sctx StateSwitchContext) error {
	l := ss.Log().With().Object("orig_state_context", sctx).Stringer("current_state", ss.State()).Logger()

	e := l.Debug().Interface("voteproof", sctx.Voteproof())
	if err := sctx.Err(); err != nil {
		e.Err(err)
	}

	e.Msg("switching state")

	if err := sctx.IsValid(nil); err != nil {
		return errors.Wrap(err, "invalid state switch context")
	}

	if sctx.FromState() == base.StateEmpty {
		sctx = sctx.SetFromState(ss.State())
	}

	if ss.State() != sctx.FromState() {
		l.Debug().Msg("current state does not match in state switch context; ignore")

		return nil
	}

	switch i, err := ss.switchStateHandover(sctx); {
	case err != nil:
		return err
	default:
		sctx = i
	}

	l = ss.Log().With().Object("state_context", sctx).Logger()

	if ss.State() == sctx.ToState() {
		if sctx.Voteproof() != nil {
			l.Debug().Msg("processing voteproof into current state")
			if err := ss.processVoteproofInternal(sctx.Voteproof()); err != nil {
				l.Debug().Err(err).Msg("failed to process voteproof into current state; ignore")
			}
		}

		return nil
	}

	var err error
	var nsctx StateSwitchContext
	switch err = ss.exitState(sctx); {
	case err == nil:
	case errors.As(err, &nsctx):
		if sctx.FromState() != nsctx.ToState() {
			l.Debug().Msg("exit to state returns state switch context; will switch state")

			return err
		}
		l.Debug().Msg("exit to state returns same from state context; ignore")
	}

	switch err = ss.enterState(sctx); {
	case err == nil:
	case errors.As(err, &nsctx):
		if sctx.ToState() == nsctx.ToState() {
			l.Error().Err(errors.Errorf("enter to state returns same to state context; ignore"))

			err = nil
		}
	}

	l.Debug().Msg("state switched")

	return err
}

func (ss *States) exitState(sctx StateSwitchContext) error {
	l := ss.Log().With().Object("state_context", sctx).Logger()

	exitFunc := EmptySwitchFunc
	f, err := ss.states[sctx.FromState()].Exit(sctx)
	if err != nil {
		return err
	}
	if f != nil {
		exitFunc = f
	}

	l.Debug().Stringer("state", sctx.FromState()).Msg("state exited")

	var nsctx StateSwitchContext
	switch err := exitFunc(); {
	case err == nil:
		return nil
	case errors.As(err, &nsctx):
		return nsctx
	default:
		l.Error().Err(err).Msg("something wrong after exit")

		return nil
	}
}

func (ss *States) enterState(sctx StateSwitchContext) error {
	l := ss.Log().With().Object("state_context", sctx).Logger()

	enterFunc := EmptySwitchFunc
	if f, err := ss.states[sctx.ToState()].Enter(sctx); err != nil {
		return err
	} else if f != nil {
		enterFunc = f
	}

	ss.setState(sctx.ToState())
	l.Debug().Stringer("state", sctx.ToState()).Msg("state entered; set current")

	var nsctx StateSwitchContext
	switch err := enterFunc(); {
	case err == nil:
		return nil
	case errors.As(err, &nsctx):
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
	l := ss.Log().With().Str("voteproof_id", voteproof.ID()).Logger()
	l.Debug().Object("voteproof", voteproof).Msg("new voteproof")

	vc := NewVoteproofChecker(ss.database, ss.suffrage, ss.nodepool, ss.LastVoteproof(), voteproof)
	_ = vc.SetLogging(ss.Logging)

	err := util.NewChecker("voteproof-checker", []util.CheckerFunc{
		vc.CheckPoint,
		vc.CheckINITVoteproofWithLocalBlock,
		vc.CheckACCEPTVoteproofProposal,
	}).Check()

	var sctx StateSwitchContext
	switch {
	case err == nil:
	case errors.Is(err, SyncByVoteproofError):
		if ss.State() == base.StateSyncing {
			err = nil
		} else {
			err = NewStateSwitchContext(ss.State(), base.StateSyncing).
				SetVoteproof(voteproof).
				SetError(err)
		}
	case errors.Is(err, util.IgnoreError):
		err = nil
	case errors.As(err, &sctx):
	default:
		return err
	}

	switch ss.State() {
	case base.StateJoining, base.StateConsensus, base.StateHandover:
		if !ss.SetLastVoteproof(voteproof) {
			return util.IgnoreError.Errorf("old voteproof received")
		}
	}

	if err == nil || sctx.ToState() == ss.State() {
		return ss.states[ss.State()].ProcessVoteproof(voteproof)
	}
	return err
}

func (ss *States) processProposal(proposal base.Proposal) error {
	ss.Lock()
	defer ss.Unlock()

	l := ss.Log().With().Stringer("seal_hash", proposal.Hash()).Logger()
	pvc, err := isaac.NewProposalValidationChecker(
		ss.database, ss.suffrage, ss.nodepool,
		proposal,
		ss.LastINITVoteproof(),
	)
	if err != nil {
		return err
	}

	_ = pvc.SetLogging(ss.Logging)

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

func (ss *States) newSealProposal(proposal base.Proposal) error {
	if ss.isNoneSuffrageNode {
		return nil
	}

	if !ss.underHandover() && proposal.Fact().Proposer().Equal(ss.nodepool.LocalNode().Address()) {
		return nil
	}

	if err := ss.validateProposal(proposal); err != nil {
		return err
	}

	if err := ss.checkBallotVoteproof(proposal); err != nil {
		return err
	}

	go ss.NewProposal(proposal)

	return nil
}

func (ss *States) newSealBallot(blt base.Ballot) error {
	if ss.isNoneSuffrageNode {
		return nil
	}

	if err := ss.validateBallot(blt); err != nil {
		return err
	}

	switch i, err := ss.voteBallot(blt); {
	case err != nil:
		return err
	case i == nil:
		return ss.checkBallotVoteproof(blt)
	default:
		ss.NewVoteproof(i)

		return nil
	}
}

func (ss *States) newOperationSeal(sl operation.Seal) error {
	// NOTE save seal
	if err := ss.database.NewOperationSeals([]operation.Seal{sl}); err != nil {
		if !errors.Is(err, util.DuplicatedError) {
			return err
		}
	}

	// NOTE none-suffrage node will broadcast operation seal to suffrage nodes
	if ss.isNoneSuffrageNode {
		go ss.broadcastOperationSealToSuffrageNodes(sl)
	}

	return nil
}

func (ss *States) validateProposal(proposal base.Proposal) error {
	pvc, err := isaac.NewProposalValidationChecker(
		ss.database, ss.suffrage, ss.nodepool,
		proposal,
		ss.LastINITVoteproof(),
	)
	if err != nil {
		return err
	}

	_ = pvc.SetLogging(ss.Logging)

	var fns []util.CheckerFunc
	if proposal.Fact().Proposer().Equal(ss.nodepool.LocalNode().Address()) {
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
		case errors.Is(err, isaac.KnownSealError):
		case errors.Is(err, util.IgnoreError):
		default:
			return err
		}

		return nil
	}
	return nil
}

func (ss *States) validateBallot(blt base.Ballot) error {
	bc := NewBallotChecker(blt, ss.LastVoteproof())
	_ = bc.SetLogging(ss.Logging)

	return util.NewChecker("ballot-validation-checker", []util.CheckerFunc{
		bc.CheckWithLastVoteproof,
	}).Check()
}

func (ss *States) voteBallot(blt base.Ballot) (base.Voteproof, error) {
	if !blt.RawFact().Stage().CanVote() {
		return nil, nil
	}

	voteproof, err := ss.ballotbox.Vote(blt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to vote")
	}

	if !voteproof.IsFinished() || voteproof.IsClosed() {
		return nil, nil
	}

	ss.Log().Debug().Stringer("ballot", blt.Hash()).Object("voteproof", voteproof).Msg("voted and new voteproof")

	return voteproof, nil
}

func (ss *States) checkBallotVoteproof(blt base.Ballot) error {
	switch ss.State() {
	case base.StateJoining, base.StateConsensus, base.StateSyncing:
	default:
		return nil
	}

	if blt.FactSign().Node().Equal(ss.nodepool.LocalNode().Address()) {
		return nil
	}

	// NOTE incoming seal should be filtered as new
	var voteproof base.Voteproof
	switch t := blt.(type) {
	case base.INITBallot:
		if blt.RawFact().Round() > 0 {
			voteproof = base.NewVoteproofSet(t.BaseVoteproof(), t.ACCEPTVoteproof())
		} else {
			voteproof = t.ACCEPTVoteproof()
		}
	case base.Proposal:
		voteproof = t.BaseVoteproof()
	case base.ACCEPTBallot:
		voteproof = t.BaseVoteproof()
	default:
		return nil
	}

	l := ss.Log().With().
		Stringer("seal_hash", blt.Hash()).
		Str("voteproof_id", voteproof.ID()).
		Str("voteproof_type", fmt.Sprintf("%T", voteproof)).
		Logger()

	lvp := ss.LastVoteproof()
	// NOTE last init voteproof is nil, it means current database is empty, so
	// will follow current state
	if lvp == nil {
		go ss.NewVoteproof(voteproof)

		return nil
	}

	if base.CompareVoteproof(voteproof, lvp) < 1 {
		l.Debug().
			Stringer("last_voteproof_height", lvp.Height()).
			Uint64("last_voteproof_round", lvp.Round().Uint64()).
			Stringer("last_voteproof_stage", lvp.Stage()).
			Stringer("ballot_voteproof_height", lvp.Height()).
			Uint64("ballot_voteproof_round", lvp.Round().Uint64()).
			Stringer("ballot_voteproof_stage", lvp.Stage()).
			Msg("old or same height voteproof received")

		return nil
	}
	l.Debug().Msg("found higher voteproof found from ballot; process voteproof")

	go ss.NewVoteproof(voteproof)

	return nil
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

			cc := NewConsensusStuckChecker(lvp, state, endure)
			err := util.NewChecker("consensus-stuck-checker", []util.CheckerFunc{
				cc.IsValidState,
				cc.IsOldLastVoteproofTime,
			}).Check()

			var changed bool
			if err != nil {
				if !stucked && !errors.Is(err, util.IgnoreError) {
					changed = true
					stucked = true
				}
			} else if stucked {
				changed = true
				stucked = false
			}

			if changed {
				l := ss.Log().With().Stringer("state", state).Dur("endure", endure).Object("last_voteproof", lvp).Logger()
				if stucked {
					l.Error().Err(err).Msg("consensus stuck")
				} else {
					l.Info().Msg("consensus released from stuck")
				}
			}
		}
	}
}

func (ss *States) broadcastOperationSealToSuffrageNodes(sl operation.Seal) {
	nodes := ss.suffrage.Nodes()
	if len(nodes) < 1 {
		return
	}

	go ss.broadcast(sl, false, func(n base.Node) bool {
		return ss.suffrage.IsInside(n.Address())
	})
}

func (ss *States) broadcast(
	sl seal.Seal,
	toLocal bool,
	filter func(base.Node) bool,
) {
	l := ss.Log().With().Stringer("seal_hash", sl.Hash()).Logger()

	l.Debug().Dict("seal", LogSeal(sl)).Bool("to_local", toLocal).Msg("broadcasting seal")
	l.Trace().Interface("seal", sl).Bool("to_local", toLocal).Msg("broadcasting seal")

	if toLocal {
		go func() {
			if err := ss.NewSeal(sl); err != nil {
				ss.Log().Error().Err(err).Stringer("seal_hash", sl.Hash()).Msg("failed to send ballot to local")
			}
		}()
	}

	// NOTE broadcast nodes of Nodepool, including suffrage nodes
	switch failed, err := ss.nodepool.Broadcast(context.Background(), sl, filter); {
	case err != nil:
		l.Error().Err(err).Msg("failed to broadcast seal")
	case len(failed) > 0:
		l.Error().Errs("failed", failed).Msg("something wrong to broadcast seal")
	default:
		l.Debug().Msg("seal broadcasted")
	}
}

func (ss *States) Handover() states.Handover {
	if ss.hd == nil {
		return nil
	}

	return ss.hd
}

func (ss *States) StartHandover() error {
	err := ss.startHandover()
	switch {
	case errors.Is(err, network.HandoverRejectedError):
		ss.Log().Debug().Err(err).Msg("trying to start handover; but rejected")
	case err != nil:
		ss.Log().Debug().Err(err).Msg("trying to start handover; but failed")
	default:
		ss.Log().Debug().Msg("handover started")
	}

	return err
}

func (ss *States) startHandover() error {
	switch t := ss.State(); {
	case t == base.StateStopped, t == base.StateBroken:
		return network.HandoverRejectedError.Errorf("node can not start handover; state=%v", t)
	case t == base.StateHandover:
		return network.HandoverRejectedError.Errorf("node is already in handover")
	case t == base.StateConsensus:
		return network.HandoverRejectedError.Errorf("node is already in consensus")
	}

	if err := ss.hd.Refresh(); err != nil {
		return fmt.Errorf("trying to start handover; but failed to refresh handover: %w", err)
	}

	switch {
	case !ss.hd.UnderHandover():
		return network.HandoverRejectedError.Errorf("not under handover")
	case ss.hd.IsReady():
		return network.HandoverRejectedError.Errorf("handover was already ready")
	default:
		ss.hd.setReady(true)

		return nil
	}
}

func (ss *States) EndHandover(ci network.ConnInfo) error {
	err := ss.endHandover(ci)
	switch {
	case errors.Is(err, network.HandoverRejectedError):
		ss.Log().Debug().Err(err).Msg("trying to end handover; but rejected")
	case err != nil:
		ss.Log().Debug().Err(err).Msg("trying to end handover; but failed")
	default:
		ss.Log().Debug().Msg("handover ended")
	}

	return err
}

func (ss *States) endHandover(ci network.ConnInfo) error {
	l := ss.Log().With().Stringer("conninfo", ci).Logger()

	ch, err := ss.hd.loadChannel(ci)
	if err != nil {
		return fmt.Errorf("failed to load channel from conninfo: %w", err)
	}

	// NOTE refresh Handover; old node will prepare handover
	if err := ss.hd.Refresh(ch); err != nil {
		return fmt.Errorf("failed to refresh handover: %w", err)
	}

	if !ss.underHandover() {
		return network.HandoverRejectedError.Errorf("after handover; handover refreshed, but not under handover")
	}

	l.Debug().Msg("trying to end handover; handover refreshed and under handover")

	go func() {
		if err := ss.timers.StopTimersAll(); err != nil {
			ss.Log().Error().Err(err).Msg("failed to stop all timers for end handover")
		}
	}()

	old := ss.hd.OldNode()

	// NOTE remove passthrough
	if err := ss.nodepool.SetPassthrough(old, func(sl network.PassthroughedSeal) bool {
		if _, ok := sl.Seal.(base.Ballot); ok { // NOTE prevent broadcasting ballots
			return false
		}

		return true
	}, 0); err != nil {
		return fmt.Errorf("failed to update passthrough from EndHandoverSeal: %w", err)
	}

	l.Debug().Msg("after handover; passthrough updated")

	// NOTE leave from discovery
	if ss.isJoined() {
		if err := ss.dis.Leave(time.Second * 3); err != nil {
			return fmt.Errorf("failed to end handover: %w", err)
		}
	}

	l.Debug().Msg("after handover; left discovery; will move to syncing")

	// NOTE switch to (only)syncing
	return ss.SwitchState(NewStateSwitchContext(base.StateEmpty, base.StateSyncing).allowEmpty(true))
}

func (ss *States) underHandover() bool {
	if ss.hd == nil {
		return false
	}

	return ss.hd.UnderHandover()
}

func (ss *States) isHandoverReady() bool {
	if ss.hd == nil {
		return false
	}

	return ss.hd.IsReady()
}

func (ss *States) join() error {
	if ss.isNoneSuffrageNode {
		return nil
	}

	if ss.hd != nil {
		if err := ss.hd.Start(); err != nil {
			return err
		}

		if ss.hd.UnderHandover() {
			ss.Log().Debug().Msg("duplicated node found; under handler; join later")

			return nil
		}

		if err := ss.joinDiscovery(3, nil); err != nil {
			ss.Log().Error().Err(err).Msg("failed to join discovery")
		}
	}

	return nil
}

func (ss *States) isJoined() bool {
	if ss.dis == nil {
		return false
	}

	return ss.dis.IsJoined()
}

func (ss *States) joinDiscovery(maxretry int, donech chan error) error {
	if ss.underHandover() {
		if old := ss.hd.OldNode(); old != nil {
			if err := ss.nodepool.SetPassthrough(old, nil, 0); err != nil {
				return fmt.Errorf("failed to set passthrough to old node: %w", err)
			}

			ss.Log().Debug().Stringer("conninfo", old.ConnInfo()).Msg("set passthrough for old node")
		}
	}

	if len(ss.suffrage.Nodes()) < 2 {
		ss.Log().Debug().Msg("empty remote suffrage nodes; skip to join discovery")

		return nil
	}

	ss.Log().Debug().Msg("trying to join discovery")

	return ss.joinDiscoveryFunc(maxretry, donech)
}

func (ss *States) defaultJoinDiscovery(maxretry int, donech chan error) error {
	if ss.dis == nil {
		return nil
	}

	switch err := ss.dis.Join(maxretry); {
	case err == nil:
	case errors.Is(err, util.IgnoreError):
		ss.Log().Error().Err(err).Msg("failed to join discovery; ignored")
	case errors.Is(err, memberlist.JoiningCanceledError):
		ss.Log().Error().Err(err).Msg("failed to join discovery; canceled")
	default:
		ss.Log().Error().Err(err).Msg("failed to join discovery")

		return err
	}

	if !ss.isJoined() {
		ss.Log().Debug().Msg("failed to join discovery; will keep trying")

		go func() {
			ss.dis.KeepTrying(context.Background(), donech)
		}()
	}

	return nil
}

func (ss *States) switchStateHandover(sctx StateSwitchContext) (StateSwitchContext, error) {
	if sctx.FromState() == base.StateHandover {
		return sctx, nil
	}

	l := ss.Log().With().Object("state_context", sctx).Logger()

	e := l.Debug().Interface("voteproof", sctx.Voteproof())

	switch {
	case sctx.FromState() == sctx.ToState():
		return sctx, nil
	case sctx.ToState() == base.StateBooting:
		return sctx, nil
	case !ss.underHandover():
		if sctx.ToState() == base.StateHandover {
			e.Msg("not under handover, to move handover state will be ignored")

			return sctx, util.IgnoreError.Errorf("handover is not ready")
		}

		return sctx, nil
	case !ss.isHandoverReady():
		e.Msg("handover not yet ready, move to syncing")

		return sctx.SetToState(base.StateSyncing), nil
	case sctx.ToState() == base.StateConsensus:
		e.Msg("under handover; consensus -> handover")

		return sctx.SetToState(base.StateHandover), nil
	default:
		return sctx, nil
	}
}
