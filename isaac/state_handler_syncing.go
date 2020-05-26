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
}

func NewStateSyncingHandler(
	localstate *Localstate,
	proposalProcessor ProposalProcessor,
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

func (ss *StateSyncingHandler) NewSeal(sl seal.Seal) error {
	switch t := sl.(type) {
	case ballot.Proposal:
		return ss.handleProposal(t)
	default:
		ss.Log().Debug().
			Hinted("seal_hint", sl.Hint()).
			Hinted("seal_hash", sl.Hash()).
			Msg("this type of Seal will be ignored")

		return nil
	}
}

func (ss *StateSyncingHandler) NewVoteproof(voteproof base.Voteproof) error {
	return ss.handleVoteproof(voteproof)
}

func (ss *StateSyncingHandler) handleProposal(proposal ballot.Proposal) error {
	l := ss.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("proposal_hash", proposal.Hash()).
			Hinted("proposal_height", proposal.Height()).
			Hinted("proposal_round", proposal.Round())
	})

	l.Debug().Msg("got proposal")

	// NOTE if proposal is the expected, process it
	var manifest block.Manifest
	if m, err := ss.localstate.Storage().LastManifest(); err != nil {
		return err
	} else {
		manifest = m
	}

	switch d := proposal.Height() - manifest.Height(); {
	case d == 1:
		ss.Log().Debug().Msg("expected proposal received; it will be processed")

		if err := ss.processProposal(proposal); err != nil {
			return err
		}
	case d > 1:
		if err := ss.newSyncerFromBallot(proposal); err != nil {
			return err
		}
	default:
		ss.Log().Debug().
			Hinted("proposal_height", proposal.Height()).
			Hinted("block_height", manifest.Height()).
			Msg("no expected proposal found")
	}

	return nil
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
			from = 0
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

func (ss *StateSyncingHandler) newSyncerFromBallot(blt ballot.Ballot) error {
	to := blt.Height() - 1

	var sourceNodes []Node
	if n, found := ss.localstate.Nodes().Node(blt.Node()); !found {
		return xerrors.Errorf("Ballot().Node() is not known node")
	} else {
		sourceNodes = append(sourceNodes, n)
	}

	ss.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var addresses []string
		for _, n := range sourceNodes {
			addresses = append(addresses, n.Address().String())
		}

		return e.Strs("source_nodes", addresses)
	}).
		Hinted("ballot", blt.Hash()).
		Hinted("height_to", to).
		Msg("will sync to the height from ballot")

	return ss.newSyncer(to, sourceNodes)
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
		if n, found := ss.localstate.Nodes().Node(address); !found {
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
		if next != nil {
			if err := next.Save(); err != nil {
				ss.Log().Error().Err(err).Msg("failed to next syncer.Save()")
			}
		}

		if len(ss.scs) < 2 {
			ss.scs = nil
		} else { // NOTE remove syncer; index can not be 0
			i := index - 1
			if i < len(ss.scs)-1 {
				copy(ss.scs[i:], ss.scs[i+1:])
			}
			ss.scs[len(ss.scs)-1] = nil
			ss.scs = ss.scs[:len(ss.scs)-1]
		}
	}
}

func (ss *StateSyncingHandler) processProposal(proposal ballot.Proposal) error {
	l := ss.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("proposal_height", proposal.Height()).
			Hinted("proposal_round", proposal.Round()).
			Hinted("proposal_hash", proposal.Hash())
	})

	iv := ss.localstate.LastINITVoteproof()
	if iv == nil {
		l.Debug().Msg("empty last INITVoteproof; proposal will be ignored")
		return nil
	}

	if iv.Height() != proposal.Height() || iv.Round() != proposal.Round() {
		l.Debug().
			Hinted("voteproof_height", iv.Height()).
			Hinted("voteproof_round", iv.Round()).
			Msg("last INITVoteproof is not for proposal; proposal will be ignored")
		return nil
	}

	if _, err := ss.proposalProcessor.ProcessINIT(proposal.Hash(), iv); err != nil {
		return err
	}

	return nil
}

func (ss *StateSyncingHandler) handleVoteproof(voteproof base.Voteproof) error {
	var baseBlock block.Manifest
	if m, err := ss.localstate.Storage().LastManifest(); err != nil {
		return err
	} else {
		baseBlock = m
	}

	l := ss.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("voteproof_stage", voteproof.Stage()).
			Hinted("voteproof_height", voteproof.Height()).
			Hinted("voteproof_round", voteproof.Round()).
			Hinted("local_height", baseBlock.Height()).
			Hinted("local_round", baseBlock.Round())
	})

	l.Debug().Msg("got voteproof")

	d := voteproof.Height() - baseBlock.Height()
	switch {
	// NOTE next voteproof of current
	case d == 1:
		switch voteproof.Stage() {
		case base.StageINIT:
			// NOTE with INIT voteproof, moves to consensus
			l.Debug().Msg("init voteproof, expected; moves to consensus")
			return ss.ChangeState(base.StateConsensus, voteproof, nil)
		case base.StageACCEPT:
			// NOTE if proposal of Voteproof is already processed, store new
			// block from Voteproof. And then will wait next INIT voteproof.
			acceptFact := voteproof.Majority().(ballot.ACCEPTBallotFact)
			if ss.proposalProcessor != nil && ss.proposalProcessor.IsProcessed(acceptFact.Proposal()) {
				l.Debug().Msg("proposal of voteproof is already processed, finish processing")

				return ss.StoreNewBlockByVoteproof(voteproof)
			}

			l.Debug().Msg("accept voteproof, ahead of local; sync")
			if err := ss.newSyncerFromVoteproof(voteproof); err != nil {
				return err
			}

			return nil
		}
	case d > 1:
		l.Debug().Msg("voteproof, ahead of local; sync")
		if err := ss.newSyncerFromVoteproof(voteproof); err != nil {
			return err
		}
	default:
		l.Debug().
			Hinted("block_height", baseBlock.Height()).
			Msg("something wrong, behind voteproof received")
	}

	return nil
}

func (ss *StateSyncingHandler) handleBallot(blt ballot.Ballot) error {
	if err := ss.newSyncerFromBallot(blt); err != nil {
		return err
	}

	return nil
}
