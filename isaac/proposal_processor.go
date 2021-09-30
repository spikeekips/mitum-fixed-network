package isaac

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
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/tree"
)

type DefaultProcessor struct {
	sync.RWMutex
	*logging.Logging
	stateLock        sync.RWMutex
	database         storage.Database
	blockData        blockdata.BlockData
	nodepool         *network.Nodepool
	baseManifest     block.Manifest
	suffrage         base.Suffrage
	oprHintset       *hint.Hintmap
	proposal         ballot.Proposal
	initVoteproof    base.Voteproof
	state            prprocessor.State
	blk              block.BlockUpdater
	suffrageInfo     block.SuffrageInfoV0
	operations       []operation.Operation
	states           []state.State
	operationsTree   tree.FixedTree
	statesTree       tree.FixedTree
	ss               storage.DatabaseSession
	blockDataSession blockdata.Session
	prePrepareHook   func(context.Context) error
	postPrepareHook  func(context.Context) error
	preSaveHook      func(context.Context) error
	postSaveHook     func(context.Context) error
	staticsLock      sync.RWMutex
	statics          map[string]interface{}
	prepareCtx       context.Context
	prepareCancel    func()
}

func NewDefaultProcessorNewFunc(
	db storage.Database,
	blockData blockdata.BlockData,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	oprHintset *hint.Hintmap,
) prprocessor.ProcessorNewFunc {
	return func(proposal ballot.Proposal, initVoteproof base.Voteproof) (prprocessor.Processor, error) {
		return NewDefaultProcessor(
			db,
			blockData,
			nodepool,
			suffrage,
			oprHintset,
			proposal,
			initVoteproof,
		)
	}
}

func NewDefaultProcessor(
	db storage.Database,
	blockData blockdata.BlockData,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	oprHintset *hint.Hintmap,
	proposal ballot.Proposal,
	initVoteproof base.Voteproof,
) (*DefaultProcessor, error) {
	var baseManifest block.Manifest
	switch m, found, err := db.ManifestByHeight(proposal.Height() - 1); {
	case err != nil:
		return nil, err
	case !found:
		return nil, util.NotFoundError.Errorf("base manifest, %d is empty", proposal.Height()-1)
	default:
		baseManifest = m
	}

	pp := &DefaultProcessor{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "default-proposal-processor").
				Int64("height", proposal.Height().Int64()).
				Uint64("round", proposal.Round().Uint64()).
				Stringer("proposal", proposal.Hash())
		}),
		database:      db,
		blockData:     blockData,
		nodepool:      nodepool,
		baseManifest:  baseManifest,
		suffrage:      suffrage,
		oprHintset:    oprHintset,
		proposal:      proposal,
		state:         prprocessor.BeforePrepared,
		initVoteproof: initVoteproof,
		preSaveHook:   nil,
		postSaveHook:  nil,
		statics: map[string]interface{}{
			"processor": "default-processor",
		},
		prepareCtx:    context.Background(),
		prepareCancel: func() {},
	}

	i, err := pp.getSuffrageInfo()
	if err != nil {
		return nil, err
	}
	pp.suffrageInfo = i

	return pp, nil
}

func (pp *DefaultProcessor) State() prprocessor.State {
	pp.stateLock.RLock()
	defer pp.stateLock.RUnlock()

	return pp.state
}

func (pp *DefaultProcessor) setState(s prprocessor.State) {
	pp.stateLock.Lock()
	defer pp.stateLock.Unlock()

	if s <= pp.state {
		return
	}

	pp.state = s
}

func (pp *DefaultProcessor) Proposal() ballot.Proposal {
	return pp.proposal
}

func (pp *DefaultProcessor) Block() block.Block {
	pp.RLock()
	defer pp.RUnlock()

	return pp.blk
}

func (pp *DefaultProcessor) setStatic(key string, value interface{}) *DefaultProcessor {
	pp.staticsLock.Lock()
	defer pp.staticsLock.Unlock()

	pp.statics[key] = value

	return pp
}

func (pp *DefaultProcessor) Statics() map[string]interface{} {
	pp.staticsLock.RLock()
	defer pp.staticsLock.RUnlock()

	return pp.statics
}

func (pp *DefaultProcessor) SetACCEPTVoteproof(acceptVoteproof base.Voteproof) error {
	pp.Lock()
	defer pp.Unlock()

	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_set_accept_voteproof_elapsed", time.Since(started))
	}()

	switch {
	case pp.blk == nil:
		return errors.Errorf("empty block, not prepared")
	case pp.ss == nil:
		return errors.Errorf("empty block session, not prepared")
	case pp.blockDataSession == nil:
		return errors.Errorf("empty block database session, not prepared")
	}

	if m := acceptVoteproof.Majority(); m == nil {
		return errors.Errorf("acceptVoteproof has empty majority")
	} else if fact, ok := m.(ballot.ACCEPTFact); !ok {
		return errors.Errorf("acceptVoteproof does not have ballot.ACCEPTBallotFact")
	} else if !pp.blk.Hash().Equal(fact.NewBlock()) {
		return errors.Errorf("hash of the processed block does not match with acceptVoteproof")
	}

	pp.blk = pp.blk.SetACCEPTVoteproof(acceptVoteproof)
	if err := pp.ss.SetACCEPTVoteproof(acceptVoteproof); err != nil {
		return err
	}

	return pp.blockDataSession.SetACCEPTVoteproof(acceptVoteproof)
}

func (pp *DefaultProcessor) Cancel() error {
	if pp.State() == prprocessor.Canceled {
		return nil
	}

	pp.Lock()
	defer pp.Unlock()

	started := time.Now()
	defer func() {
		_ = pp.setStatic("processor_cancel_elapsed", time.Since(started))
	}()

	pp.prepareCancel()

	if err := pp.resetPrepare(); err != nil {
		return err
	}

	pp.setState(prprocessor.Canceled)

	return nil
}

func (pp *DefaultProcessor) BaseManifest() block.Manifest {
	return pp.baseManifest
}

func (pp *DefaultProcessor) SuffrageInfo() block.SuffrageInfoV0 {
	return pp.suffrageInfo
}

func (pp *DefaultProcessor) getSuffrageInfo() (block.SuffrageInfoV0, error) {
	var ns []base.Node // nolint:prealloc
	for _, address := range pp.suffrage.Nodes() {
		n, _, found := pp.nodepool.Node(address)
		if !found {
			return block.SuffrageInfoV0{}, errors.Errorf("suffrage node, %s not found in node pool", address)
		}
		ns = append(ns, n)
	}

	return block.NewSuffrageInfoV0(pp.proposal.Node(), ns), nil
}
