package isaac

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
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
	ss := &StateSyncingHandler{
		BaseStateHandler: NewBaseStateHandler(localstate, proposalProcessor, StateSyncing),
		stateChan:        make(chan Syncer),
	}
	ss.BaseStateHandler.Logging = logging.NewLogging(func(c zerolog.Context) zerolog.Context {
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
	case ctx.Ballot() != nil:
		if err := ss.handleBallot(ctx.Ballot()); err != nil {
			return err
		}
	case ctx.Voteproof() != nil:
		if err := ss.handleVoteproof(ctx.Voteproof()); err != nil {
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
	case Proposal:
		return ss.handleProposal(t)
	default:
		ss.Log().Debug().
			Str("seal_hint", sl.Hint().Verbose()).
			Str("seal_hash", sl.Hash().String()).
			Msg("this type of Seal will be ignored")

		return nil
	}
}

func (ss *StateSyncingHandler) NewVoteproof(voteproof Voteproof) error {
	return ss.handleVoteproof(voteproof)
}

func (ss *StateSyncingHandler) handleProposal(proposal Proposal) error {
	l := ss.Log().WithLogger(func(ctx zerolog.Context) zerolog.Context {
		return ctx.Str("proposal_hash", proposal.Hash().String()).
			Int64("proposal_height", proposal.Height().Int64()).
			Uint64("proposal_round", proposal.Round().Uint64())
	})

	l.Debug().Msg("got proposal")

	// NOTE if proposal is the expected, process it
	base := ss.localstate.LastBlock().Height()
	switch d := proposal.Height() - base; {
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
			Int64("proposal_height", proposal.Height().Int64()).
			Int64("block_height", base.Int64()).
			Msg("no expected proposal found")
	}

	return nil
}

func (ss *StateSyncingHandler) newSyncer(to Height, sourceNodes []Node) error {
	ss.Lock()
	defer ss.Unlock()

	lastBlock := ss.localstate.LastBlock()

	var lastSyncer Syncer
	var from Height
	if len(ss.scs) < 1 {
		if lastBlock == nil {
			from = 0
		} else {
			from = lastBlock.Height() + 1
		}
	} else {
		lastSyncer = ss.scs[len(ss.scs)-1]
		from = lastSyncer.HeightTo() + 1
	}

	if lastSyncer != nil && to <= lastSyncer.HeightTo() {
		ss.Log().Debug().Int64("height_to", to.Int64()).Msg("already started to sync")
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
		if err := syncer.Prepare(lastBlock.Manifest()); err != nil {
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

func (ss *StateSyncingHandler) newSyncerFromBallot(ballot Ballot) error {
	to := ballot.Height() - 1

	var sourceNodes []Node
	if n, found := ss.localstate.Nodes().Node(ballot.Node()); !found {
		return xerrors.Errorf("Ballot().Node() is not known node")
	} else {
		sourceNodes = append(sourceNodes, n)
	}

	ss.Log().VerboseFunc(func(e *zerolog.Event) *zerolog.Event {
		var addresses []string
		for _, n := range sourceNodes {
			addresses = append(addresses, n.Address().String())
		}

		return e.Strs("source_nodes", addresses)
	}).
		Str("ballot", ballot.Hash().String()).
		Int64("height_to", to.Int64()).
		Msg("will sync to the height from ballot")

	return ss.newSyncer(to, sourceNodes)
}

func (ss *StateSyncingHandler) newSyncerFromVoteproof(voteproof Voteproof) error {
	var to Height
	switch voteproof.Stage() {
	case StageINIT:
		to = voteproof.Height() - 1
	case StageACCEPT:
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

	ss.Log().VerboseFunc(func(e *zerolog.Event) *zerolog.Event {
		var addresses []string
		for _, n := range sourceNodes {
			addresses = append(addresses, n.Address().String())
		}

		return e.Strs("source_nodes", addresses)
	}).
		Int64("voteproof_height", voteproof.Height().Int64()).
		Uint64("voteproof_round", voteproof.Round().Uint64()).
		Int64("height_to", to.Int64()).
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

		var block Block
		if err := util.Retry(3, time.Millisecond*300, func() error {
			if b, err := ss.localstate.Storage().LastBlock(); err != nil {
				return err
			} else {
				block = b
			}

			return nil
		}); err != nil {
			ss.Log().Error().Err(err).Msg("failed to get last block after synced")
			return
		}

		if err := ss.localstate.SetLastBlock(block); err != nil {
			ss.Log().Error().Err(err).Msg("failed to set last block after synced")
		}
	}
}

func (ss *StateSyncingHandler) processProposal(proposal Proposal) error {
	l := ss.Log().WithLogger(func(ctx zerolog.Context) zerolog.Context {
		return ctx.Int64("proposal_height", proposal.Height().Int64()).
			Uint64("proposal_round", proposal.Round().Uint64()).
			Str("proposal_hash", proposal.Hash().String())
	})

	iv := ss.localstate.LastINITVoteproof()
	if iv == nil {
		l.Debug().Msg("empty last INITVoteproof; proposal will be ignored")
		return nil
	}

	if iv.Height() != proposal.Height() || iv.Round() != proposal.Round() {
		l.Debug().
			Int64("voteproof_height", iv.Height().Int64()).
			Uint64("voteproof_round", iv.Round().Uint64()).
			Msg("last INITVoteproof is not for proposal; proposal will be ignored")
		return nil
	}

	if _, err := ss.proposalProcessor.ProcessINIT(proposal.Hash(), iv); err != nil {
		return err
	}

	return nil
}

func (ss *StateSyncingHandler) handleVoteproof(voteproof Voteproof) error {
	base := ss.localstate.LastBlock()

	l := ss.Log().WithLogger(func(ctx zerolog.Context) zerolog.Context {
		return ctx.Str("voteproof_stage", voteproof.Stage().String()).
			Int64("voteproof_height", voteproof.Height().Int64()).
			Uint64("voteproof_round", voteproof.Round().Uint64()).
			Int64("local_height", base.Height().Int64()).
			Uint64("local_round", base.Round().Uint64())
	})

	l.Debug().Msg("got voteproof")

	d := voteproof.Height() - base.Height()
	switch {
	// NOTE next voteproof of current
	case d == 1:
		switch voteproof.Stage() {
		case StageINIT:
			// NOTE with INIT voteproof, moves to consensus
			l.Debug().Msg("init voteproof, expected; moves to consensus")
			return ss.ChangeState(StateConsensus, voteproof, nil)
		case StageACCEPT:
			// NOTE if proposal of Voteproof is already processed, store new
			// block from Voteproof. And then will wait next INIT voteproof.
			acceptFact := voteproof.Majority().(ACCEPTBallotFact)
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
			Int64("block_height", base.Height().Int64()).
			Msg("something wrong, behind voteproof received")
	}

	return nil
}

func (ss *StateSyncingHandler) handleBallot(ballot Ballot) error {
	if err := ss.newSyncerFromBallot(ballot); err != nil {
		return err
	}

	return nil
}
