package isaac

import (
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
)

const TimerIDWaitVoteproof = "wait-voteproof-for-syncing-to-joining"

/*
StateSyncingHandler will sync the block states to the latest. Usually state is transited to syncing,

* newly accepted Voteproof is ahead of local state
* without Voteproof and insufficient voting received, valid incoming INIT or
 ACCEPT ballot is ahead of local state

Basically syncing handler tries to find the source nodes at first. The source
nodes will be selected by their latest activies,

* if handler is activated by voteproof, the ballot nodes will be source nodes
* if handler is activated by ballot, the ballot node will be source node.

With the target height, handler will start to sync up to target height and then
will wait proposal, which is the next of the synced block. Handler will keep
syncing and processing proposal until INIT Voteproof is received. If no INIT
Voteproof received within a given time, states will be changed to joining state.
*/
type StateSyncingHandler struct {
	sync.RWMutex
	*BaseStateHandler
	lv                   base.Voteproof
	lvLock               sync.RWMutex
	syncs                *Syncers
	waitVoteproofTimeout time.Duration
}

func NewStateSyncingHandler(localstate *Localstate) *StateSyncingHandler {
	ss := &StateSyncingHandler{
		BaseStateHandler:     NewBaseStateHandler(localstate, nil, base.StateSyncing),
		waitVoteproofTimeout: time.Second * 5, // NOTE long enough time
	}
	ss.BaseStateHandler.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
		return c.Str("module", "consensus-state-syncing-handler")
	})
	ss.timers = localtime.NewTimers([]string{TimerIDWaitVoteproof}, false)

	return ss
}

func (ss *StateSyncingHandler) syncers() *Syncers {
	ss.RLock()
	defer ss.RUnlock()

	return ss.syncs
}

func (ss *StateSyncingHandler) newSyncers() error {
	if syncs := ss.syncs; syncs != nil {
		if !syncs.isFinished() {
			return xerrors.Errorf("syncers still running")
		} else if err := syncs.Stop(); err != nil {
			if !xerrors.Is(err, util.DaemonAlreadyStoppedError) {
				return err
			}
		}
	}

	baseHeight := base.NilHeight
	var baseManifest block.Manifest
	if m, found, err := ss.localstate.Storage().LastManifest(); err != nil {
		return err
	} else if found {
		baseHeight = m.Height()
		baseManifest = m
	}

	syncs := NewSyncers(ss.localstate, baseManifest)
	syncs.WhenFinished(ss.whenFinished)
	syncs.WhenBlockSaved(ss.whenBlockSaved)
	_ = syncs.SetLogger(ss.Log())

	if err := syncs.Start(); err != nil {
		return err
	}

	ss.Log().Debug().Hinted("base_height", baseHeight).Msg("new syncers started")

	ss.syncs = syncs

	return nil
}

func (ss *StateSyncingHandler) SetLogger(l logging.Logger) logging.Logger {
	_ = ss.Logging.SetLogger(l)
	_ = ss.timers.SetLogger(l)

	return ss.Log()
}

func (ss *StateSyncingHandler) Activate(ctx *StateChangeContext) error {
	ss.Lock()
	defer ss.Unlock()

	if ctx == nil {
		return xerrors.Errorf("empty StateChangeContext")
	}

	switch {
	case ctx.Voteproof() != nil:
	case ctx.Ballot() != nil:
	case ctx.From() == base.StateBooting:
	default:
		return xerrors.Errorf("empty voteproof or ballot in StateChangeContext")
	}

	ss.BaseStateHandler.activate()

	if err := ss.newSyncers(); err != nil {
		return err
	}

	go func() {
		if err := ss.activate(ctx); err != nil {
			ss.Log().Error().Err(err).Msg("failed to activate syncing handler")
		}
	}()

	ss.Log().Debug().Msg("activated")

	return nil
}

func (ss *StateSyncingHandler) Deactivate(_ *StateChangeContext) error {
	ss.Log().Debug().Msg("deactivated")

	ss.deactivate()

	ss.lvLock.Lock()
	ss.lv = nil
	ss.lvLock.Unlock()

	if syncs := ss.syncers(); syncs != nil {
		if err := syncs.Stop(); err != nil {
			return err
		}
	}

	if err := ss.timers.Stop(); err != nil {
		return xerrors.Errorf("failed to stop timers of syncing handler: %w", err)
	}

	return nil
}

func (ss *StateSyncingHandler) activate(ctx *StateChangeContext) error {
	switch {
	case ctx.Voteproof() != nil:
		if err := ss.handleVoteproof(ctx.Voteproof()); err != nil {
			return err
		}
	case ctx.Ballot() != nil:
		if err := ss.handleBallot(ctx.Ballot()); err != nil {
			return err
		}
	case ctx.From() == base.StateBooting:
		ss.Log().Debug().Msg("syncing started from booting without initial block")
	default:
		return xerrors.Errorf("empty voteproof or ballot in StateChangeContext")
	}

	return nil
}

func (ss *StateSyncingHandler) NewSeal(seal.Seal) error {
	return nil
}

func (ss *StateSyncingHandler) NewVoteproof(voteproof base.Voteproof) error {
	return ss.handleVoteproof(voteproof)
}

func (ss *StateSyncingHandler) fromVoteproof(voteproof base.Voteproof) error {
	var to base.Height
	switch voteproof.Stage() {
	case base.StageINIT:
		to = voteproof.Height() - 1
	case base.StageACCEPT:
		to = voteproof.Height()
	default:
		return xerrors.Errorf("invalid Voteproof received")
	}

	var sourceNodes []network.Node
	for i := range voteproof.Votes() {
		nf := voteproof.Votes()[i]
		if ss.localstate.Node().Address().Equal(nf.Node()) {
			continue
		} else if n, found := ss.localstate.Nodes().Node(nf.Node()); !found {
			return xerrors.Errorf("node in Voteproof is not known node")
		} else {
			sourceNodes = append(sourceNodes, n)
		}
	}

	ss.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var addresses []string
		for _, n := range sourceNodes {
			addresses = append(addresses, n.Address().String())
		}

		return e.Strs("source_nodes", addresses)
	}).
		Hinted("voteproof_height", voteproof.Height()).
		Hinted("voteproof_round", voteproof.Round()).
		Hinted("height_to", to).
		Msg("will sync to the height")

	return ss.syncers().Add(to, sourceNodes)
}

func (ss *StateSyncingHandler) handleVoteproof(voteproof base.Voteproof) error {
	baseHeight := base.PreGenesisHeight
	if m, found, err := ss.localstate.Storage().LastManifest(); err != nil {
		return err
	} else if found {
		baseHeight = m.Height()
	}

	l := ss.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("voteproof_stage", voteproof.Stage()).
			Hinted("voteproof_height", voteproof.Height()).
			Hinted("voteproof_round", voteproof.Round()).
			Hinted("local_height", baseHeight)
	})

	l.Debug().Msg("got voteproof for syncing")

	var to base.Height
	if h, err := ss.getExpectedHeightFromoteproof(voteproof); err != nil {
		return err
	} else {
		to = h
	}

	if d := to - baseHeight; d < 0 {
		return nil
	} else if d > 0 {
		l.Debug().Msg("voteproof, ahead of local; sync")

		if err := ss.timers.StopTimers([]string{TimerIDWaitVoteproof}); err != nil {
			l.Error().Err(err).Str("timer", TimerIDWaitVoteproof).Msg("failed to stop")
		}

		return ss.fromVoteproof(voteproof)
	}

	if voteproof.Stage() != base.StageINIT {
		return nil
	}

	l.Debug().Msg("init voteproof, expected")

	if !ss.syncers().isFinished() {
		l.Debug().Msg("init voteproof, expected; but syncing is not finished")

		return nil
	}

	if err := ss.timers.StopTimers([]string{TimerIDWaitVoteproof}); err != nil {
		ss.Log().Error().Err(err).Str("timer", TimerIDWaitVoteproof).Msg("failed to stop")
	}

	l.Debug().Msg("init voteproof, expected; moves to consensus")

	return ss.ChangeState(base.StateConsensus, voteproof, nil)
}

func (ss *StateSyncingHandler) handleBallot(blt ballot.Ballot) error {
	var voteproof base.Voteproof
	switch t := blt.(type) {
	case ballot.Proposal:
		ss.Log().Debug().Hinted("seal_hash", blt.Hash()).Msg("ignore proposal ballot for syncing")
		return nil
	case ballot.INITBallot:
		voteproof = t.Voteproof()
	case ballot.ACCEPTBallot:
		voteproof = t.Voteproof()
	}

	return ss.fromVoteproof(voteproof)
}

func (ss *StateSyncingHandler) getExpectedHeightFromoteproof(voteproof base.Voteproof) (base.Height, error) {
	switch voteproof.Stage() {
	case base.StageINIT:
		return voteproof.Height() - 1, nil
	case base.StageACCEPT:
		return voteproof.Height(), nil
	default:
		return base.NilHeight, xerrors.Errorf("invalid Voteproof received")
	}
}

func (ss *StateSyncingHandler) whenFinished(height base.Height) {
	ss.Log().Debug().Hinted("height", height).Msg("syncing finished; start timer")

	if timer, err := ss.timerWaitVoteproof(); err != nil {
		ss.Log().Error().Err(err).Str("timer", TimerIDWaitVoteproof).Msg("failed to make timer")

		return
	} else if err := ss.timers.SetTimer(TimerIDWaitVoteproof, timer); err != nil {
		ss.Log().Error().Err(err).Str("timer", TimerIDWaitVoteproof).Msg("failed to set timer")

		return
	}

	if err := ss.timers.StartTimers([]string{TimerIDWaitVoteproof}, true); err != nil {
		ss.Log().Error().Err(err).Str("timer", TimerIDWaitVoteproof).Msg("failed to start timer")

		return
	}
}

func (ss *StateSyncingHandler) timerWaitVoteproof() (*localtime.CallbackTimer, error) {
	return localtime.NewCallbackTimer(
		TimerIDWaitVoteproof,
		func() (bool, error) {
			ss.Log().Debug().Msg("syncing finished, but no more Voteproof; moves to joining state")
			if err := ss.ChangeState(base.StateJoining, nil, nil); err != nil {
				return false, err
			}

			return false, nil
		},
		ss.waitVoteproofTimeout,
		nil,
	)
}
