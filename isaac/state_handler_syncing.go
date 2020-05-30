package isaac

import (
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/logging"
)

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
	scs       []Syncer
	stateChan chan Syncer
	lv        base.Voteproof
}

func NewStateSyncingHandler(
	localstate *Localstate,
	proposalProcessor ProposalProcessor, // BLOCK remove
) (*StateSyncingHandler, error) {
	// TODO if already synced and no voteproof, should go to the consensus state.
	ss := &StateSyncingHandler{
		BaseStateHandler: NewBaseStateHandler(localstate, proposalProcessor, base.StateSyncing),
		stateChan:        make(chan Syncer),
	}
	ss.BaseStateHandler.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
		return c.Str("module", "consensus-state-syncing-handler")
	})

	return ss, nil
}

func (ss *StateSyncingHandler) Activate(ctx StateChangeContext) error {
	l := loggerWithStateChangeContext(ctx, ss.Log())
	l.Debug().Msg("activated")

	go func() {
		for syncer := range ss.stateChan {
			ss.syncerStateChanged(syncer)
		}
	}()

	if m, err := ss.localstate.Storage().LastBlock(); err != nil {
		if !xerrors.Is(err, storage.NotFoundError) {
			return err
		}
	} else {
		ss.setLastVoteproof(m.ACCEPTVoteproof())
	}

	// TODO also compare the hash of target block with height

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
		ss.Log().Debug().Msg("syncing started from booting wihout initial block")
	default:
		return xerrors.Errorf("empty voteproof or ballot in StateChangeContext")
	}

	return nil
}

func (ss *StateSyncingHandler) Deactivate(ctx StateChangeContext) error {
	l := loggerWithStateChangeContext(ctx, ss.Log())
	l.Debug().Msg("deactivated")

	return nil
}

func (ss *StateSyncingHandler) NewSeal(seal.Seal) error {
	return nil
}

func (ss *StateSyncingHandler) NewVoteproof(voteproof base.Voteproof) error {
	return ss.handleVoteproof(voteproof)
}

func (ss *StateSyncingHandler) newSyncer(to base.Height, sourceNodes []Node) error {
	ss.Lock()
	defer ss.Unlock()

	var lastManifest block.Manifest
	if m, err := ss.localstate.Storage().LastManifest(); err != nil {
		if !xerrors.Is(err, storage.NotFoundError) {
			return err
		}
	} else {
		lastManifest = m
	}

	var lastSyncer Syncer
	var from base.Height
	if len(ss.scs) < 1 {
		if lastManifest == nil {
			from = base.PreGenesisHeight
		} else {
			from = lastManifest.Height() + 1
		}
	} else {
		lastSyncer = ss.scs[len(ss.scs)-1]
		from = lastSyncer.HeightTo() + 1
	}

	if lastSyncer != nil && to <= lastSyncer.HeightTo() {
		ss.Log().Debug().Hinted("height_to", to).Msg("already started to sync")
		return nil
	}

	var syncer Syncer
	if s, err := NewGeneralSyncer(ss.localstate, sourceNodes, from, to); err != nil {
		return err
	} else {
		syncer = s.SetStateChan(ss.stateChan)
	}

	if l, ok := syncer.(logging.SetLogger); ok {
		_ = l.SetLogger(ss.Log())
	}

	if lastSyncer == nil {
		if err := syncer.Prepare(lastManifest); err != nil {
			return err
		}
	} else {
		if lastSyncer.State() >= SyncerPrepared {
			if err := syncer.Prepare(lastSyncer.TailManifest()); err != nil {
				return err
			}
		}
	}

	ss.scs = append(ss.scs, syncer)

	return nil
}

func (ss *StateSyncingHandler) newSyncerFromVoteproof(voteproof base.Voteproof) error {
	var to base.Height
	switch voteproof.Stage() {
	case base.StageINIT:
		to = voteproof.Height() - 1
	case base.StageACCEPT:
		to = voteproof.Height()
	default:
		return xerrors.Errorf("invalid Voteproof received")
	}

	var sourceNodes []Node
	for address := range voteproof.Ballots() {
		if ss.localstate.Node().Address().Equal(address) {
			continue
		} else if n, found := ss.localstate.Nodes().Node(address); !found {
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

	return ss.newSyncer(to, sourceNodes)
}

func (ss *StateSyncingHandler) nextSyncer(syncer Syncer) (int, Syncer) {
	var index int
	var next Syncer
	for i := range ss.scs {
		n := ss.scs[i]
		if n.HeightFrom() == syncer.HeightFrom() {
			continue
		}

		index = i
		next = n
		break
	}

	return index, next
}

func (ss *StateSyncingHandler) syncerStateChanged(syncer Syncer) {
	ss.Lock()
	defer ss.Unlock()

	switch syncer.State() {
	case SyncerPrepared:
		// after syncer is prepared
		// - do Save()
		// - the next syncer will do Prepare()

		go func() {
			if err := syncer.Save(); err != nil {
				ss.Log().Error().Err(err).Msg("failed to syncer.Save()")
			}
		}()

		if _, next := ss.nextSyncer(syncer); next != nil {
			if err := next.Prepare(syncer.TailManifest()); err != nil {
				ss.Log().Error().Err(err).Msg("failed to next syncer.Prepare()")
			}
		}
	case SyncerSaved:
		// after syncer saves blocks,
		// - the next syncer will do Save()
		// - remove syncer
		index, next := ss.nextSyncer(syncer)
		ss.Log().Debug().
			Int("scs", len(ss.scs)).
			Int("index", index).
			Bool("has_next", next != nil).
			Msg("trying to find next syncer")

		if len(ss.scs) < 2 {
			ss.scs = nil
		} else if index > 0 { // NOTE remove previous syncer
			i := index - 1
			if i < len(ss.scs)-1 {
				copy(ss.scs[i:], ss.scs[i+1:])
			}
			ss.scs[len(ss.scs)-1] = nil
			ss.scs = ss.scs[:len(ss.scs)-1]
		}

		if next != nil {
			if err := next.Save(); err != nil {
				ss.Log().Error().Err(err).Msg("failed to next syncer.Save()")
			}
		}
	}
}

func (ss *StateSyncingHandler) handleVoteproof(voteproof base.Voteproof) error {
	baseHeight := base.PreGenesisHeight
	if m, err := ss.localstate.Storage().LastManifest(); err != nil {
		if !xerrors.Is(err, storage.NotFoundError) {
			return err
		}
	} else {
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

	// NOTE old voteproof should be ignored
	if lv := ss.lastVoteproof(); lv != nil && to <= lv.Height() {
		if to != lv.Height() || voteproof.Stage() != base.StageINIT {
			return xerrors.Errorf("known voteproof received: height=%v", voteproof.Height())
		}
	} else if lv != nil {
		ss.setLastVoteproof(voteproof)
	}

	d := to - baseHeight
	switch {
	case d == 0:
		l.Debug().Msg("init voteproof, expected; moves to consensus")
		return ss.ChangeState(base.StateConsensus, voteproof, nil)
	default:
		l.Debug().Msg("voteproof, ahead of local; sync")
		if err := ss.newSyncerFromVoteproof(voteproof); err != nil {
			return err
		}
	}

	return nil
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

	return ss.newSyncerFromVoteproof(voteproof)
}

func (ss *StateSyncingHandler) lastVoteproof() base.Voteproof {
	ss.RLock()
	defer ss.RUnlock()

	return ss.lv
}

func (ss *StateSyncingHandler) setLastVoteproof(voteproof base.Voteproof) {
	ss.Lock()
	defer ss.Unlock()

	if ss.lv != nil && ss.lv.Height() <= voteproof.Height() {
		return
	}

	ss.Log().Debug().
		Hinted("voteproof_stage", voteproof.Stage()).
		Hinted("voteproof_height", voteproof.Height()).
		Hinted("voteproof_round", voteproof.Round()).
		Msg("new last voteproof")

	ss.lv = voteproof
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
