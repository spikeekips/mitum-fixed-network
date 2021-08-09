package basicstates

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
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
	database           storage.Database
	policy             *isaac.LocalPolicy
	nodepool           *network.Nodepool
	suffrage           base.Suffrage
	statelock          sync.RWMutex
	state              base.State
	states             map[base.State]State
	statech            chan StateSwitchContext
	voteproofch        chan base.Voteproof
	proposalch         chan ballot.Proposal
	ballotbox          *isaac.Ballotbox
	timers             *localtime.Timers
	lvplock            sync.RWMutex
	lvp                base.Voteproof
	livp               base.Voteproof
	blockSavedHook     *pm.Hooks
	isNoneSuffrageNode bool
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
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
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
	case ballot.Proposal:
		err = ss.newSealProposal(t)
	case ballot.Ballot:
		err = ss.newSealBallot(t)
	default:
		err = ss.newSealOthers(sl)
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

func (ss *States) NewProposal(proposal ballot.Proposal) {
	go func() {
		ss.proposalch <- proposal
	}()
}

// BroadcastBallot broadcast seal to the known nodes,
// - suffrage nodes
// - if toLocal is true, sends to local
func (ss *States) BroadcastBallot(blt ballot.Ballot, toLocal bool) error {
	return ss.broadcast(blt, toLocal, func(node base.Node) bool {
		return ss.suffrage.IsInside(node.Address())
	})
}

// BroadcastSeals broadcast seal to the known nodes,
// - suffrage nodes
// - and other nodes
// - if toLocal is true, sends to local
func (ss *States) BroadcastSeals(sl seal.Seal, toLocal bool) error {
	return ss.broadcast(sl, toLocal, func(base.Node) bool {
		return true
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
	ss.Log().Debug().Bool("is_none_suffrage", ss.isNoneSuffrageNode).Msg("states started")

	if ss.ballotbox != nil {
		go ss.cleanBallotbox(ctx)
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
	l := ss.Log().With().Object("state_context", sctx).Logger()

	e := l.Debug().Interface("voteproof", sctx.Voteproof())
	if err := sctx.Err(); err != nil {
		e.Err(err)
	}

	e.Msg("switching state")

	if err := sctx.IsValid(nil); err != nil {
		return errors.Wrap(err, "invalid state switch context")
	} else if ss.State() != sctx.FromState() {
		l.Debug().Msg("current state does not match in state switch context; ignore")

		return nil
	}

	if ss.State() == sctx.ToState() {
		if sctx.Voteproof() == nil {
			return errors.Errorf("same state, but empty voteproof")
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
		err = NewStateSwitchContext(ss.State(), base.StateSyncing).
			SetVoteproof(voteproof).
			SetError(err)
	case errors.Is(err, util.IgnoreError):
		err = nil
	case errors.As(err, &sctx):
	default:
		return err
	}

	switch ss.State() {
	case base.StateJoining, base.StateConsensus:
		if !ss.SetLastVoteproof(voteproof) {
			return util.IgnoreError.Errorf("old voteproof received")
		}
	}

	if err == nil || sctx.ToState() == ss.State() {
		return ss.states[ss.State()].ProcessVoteproof(voteproof)
	}
	return err
}

func (ss *States) processProposal(proposal ballot.Proposal) error {
	ss.Lock()
	defer ss.Unlock()

	l := ss.Log().With().Stringer("seal_hash", proposal.Hash()).Logger()
	pvc := isaac.NewProposalValidationChecker(
		ss.database, ss.suffrage, ss.nodepool,
		proposal,
		ss.LastINITVoteproof(),
	)
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

func (ss *States) newSealProposal(proposal ballot.Proposal) error {
	if ss.isNoneSuffrageNode {
		return nil
	}

	if proposal.Node().Equal(ss.nodepool.LocalNode().Address()) {
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

func (ss *States) newSealBallot(blt ballot.Ballot) error {
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

func (ss *States) newSealOthers(sl seal.Seal) error {
	// NOTE save seal
	if err := ss.database.NewSeals([]seal.Seal{sl}); err != nil {
		if !errors.Is(err, util.DuplicatedError) {
			return err
		}

		return err
	}

	// NOTE none-suffrage node will broadcast operation seal to suffrage nodes
	if ss.isNoneSuffrageNode {
		if i, ok := sl.(operation.Seal); ok {
			go ss.broadcastOperationSealToSuffrageNodes(i)
		}
	}

	return nil
}

func (ss *States) validateProposal(proposal ballot.Proposal) error {
	pvc := isaac.NewProposalValidationChecker(
		ss.database, ss.suffrage, ss.nodepool,
		proposal,
		ss.LastINITVoteproof(),
	)
	_ = pvc.SetLogging(ss.Logging)

	var fns []util.CheckerFunc
	if proposal.Node().Equal(ss.nodepool.LocalNode().Address()) {
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

func (ss *States) validateBallot(blt ballot.Ballot) error {
	bc := NewBallotChecker(blt, ss.LastVoteproof())
	_ = bc.SetLogging(ss.Logging)

	return util.NewChecker("ballot-validation-checker", []util.CheckerFunc{
		bc.CheckWithLastVoteproof,
	}).Check()
}

func (ss *States) voteBallot(blt ballot.Ballot) (base.Voteproof, error) {
	if !blt.Stage().CanVote() {
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

func (ss *States) checkBallotVoteproof(blt ballot.Ballot) error {
	switch ss.State() {
	case base.StateJoining, base.StateConsensus, base.StateSyncing:
	default:
		return nil
	}

	if blt.Node().Equal(ss.nodepool.LocalNode().Address()) {
		return nil
	}

	// NOTE incoming seal should be filtered as new
	var voteproof base.Voteproof
	switch t := blt.(type) {
	case ballot.INIT:
		voteproof = t.ACCEPTVoteproof()
	case ballot.Proposal:
		voteproof = t.Voteproof()
	case ballot.ACCEPT:
		voteproof = t.Voteproof()
	default:
		return nil
	}

	l := ss.Log().With().Stringer("seal_hash", blt.Hash()).Str("voteproof_id", voteproof.ID()).Logger()

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

			e := ss.Log().With().Stringer("state", state).Dur("endure", endure)

			if lvp != nil {
				e = e.Object("last_voteproof", lvp)
			}

			l := e.Logger()

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

	if err := ss.broadcast(sl, false, func(node base.Node) bool {
		return ss.suffrage.IsInside(node.Address())
	}); err != nil {
		ss.Log().Error().Err(err).Msg("problem to broadcast operation.Seal to suffrage nodes")
	}
}

func (ss *States) broadcast(
	sl seal.Seal,
	toLocal bool,
	filter func(node base.Node) bool,
) error {
	l := ss.Log().With().Stringer("seal_hash", sl.Hash()).Logger()

	l.Debug().Dict("seal", LogSeal(sl)).Bool("to_local", toLocal).Msg("broadcasting seal")
	l.Trace().Interface("seal", sl).Bool("to_local", toLocal).Msg("broadcasting seal")

	if toLocal {
		go func() {
			if err := ss.NewSeal(sl); err != nil {
				l.Error().Err(err).Msg("failed to send ballot to local")
			}
		}()
	}

	if ss.nodepool.LenRemoteAlives() < 1 {
		return nil
	}

	// NOTE broadcast nodes of Nodepool, including suffrage nodes
	var targets int
	ss.nodepool.TraverseAliveRemotes(func(no base.Node, ch network.Channel) bool {
		if !filter(no) {
			return true
		}

		targets++

		go func(no base.Node, ch network.Channel) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			if err := ch.SendSeal(ctx, sl); err != nil {
				l.Error().Err(err).Stringer("target_node", no.Address()).Msg("failed to broadcast")
			}
		}(no, ch)

		return true
	})

	l.Debug().Msg("seal broadcasted")

	return nil
}
