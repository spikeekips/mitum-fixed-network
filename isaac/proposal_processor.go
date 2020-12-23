package isaac

import (
	"context"
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/tree"
	"golang.org/x/xerrors"
)

type DefaultProcessor struct {
	sync.RWMutex
	*logging.Logging
	stateLock       sync.RWMutex
	local           base.Node
	st              storage.Storage
	blockFS         *storage.BlockFS
	nodepool        *network.Nodepool
	baseManifest    block.Manifest
	suffrage        base.Suffrage
	oprHintset      *hint.Hintmap
	proposal        ballot.Proposal
	initVoteproof   base.Voteproof
	state           prprocessor.State
	blk             block.BlockUpdater
	suffrageInfo    block.SuffrageInfoV0
	operations      []operation.Operation
	states          []state.State
	operationsTree  tree.FixedTree
	statesTree      tree.FixedTree
	bs              storage.BlockStorage
	prePrepareHook  func(context.Context) error
	postPrepareHook func(context.Context) error
	preSaveHook     func(context.Context) error
	postSaveHook    func(context.Context) error
}

func NewDefaultProcessorNewFunc(
	local base.Node,
	st storage.Storage,
	blockFS *storage.BlockFS,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	oprHintset *hint.Hintmap,
) prprocessor.ProcessorNewFunc {
	return func(proposal ballot.Proposal, initVoteproof base.Voteproof) (prprocessor.Processor, error) {
		var baseManifest block.Manifest
		switch m, found, err := st.ManifestByHeight(proposal.Height() - 1); {
		case !found:
			return nil, storage.NotFoundError.Errorf("base manifest, %d is empty", proposal.Height()-1)
		case err != nil:
			return nil, err
		default:
			baseManifest = m
		}

		pp := &DefaultProcessor{
			Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
				return c.Str("module", "default-proposal-processor").
					Hinted("height", proposal.Height()).
					Hinted("round", proposal.Round()).
					Hinted("proposal", proposal.Hash())
			}),
			local:         local,
			st:            st,
			blockFS:       blockFS,
			nodepool:      nodepool,
			baseManifest:  baseManifest,
			suffrage:      suffrage,
			oprHintset:    oprHintset,
			proposal:      proposal,
			state:         prprocessor.BeforePrepared,
			initVoteproof: initVoteproof,
			preSaveHook:   nil,
			postSaveHook:  nil,
		}

		if i, err := pp.getSuffrageInfo(); err != nil {
			return nil, err
		} else {
			pp.suffrageInfo = i
		}

		return pp, nil
	}
}

func (pp *DefaultProcessor) State() prprocessor.State {
	pp.stateLock.RLock()
	defer pp.stateLock.RUnlock()

	return pp.state
}

func (pp *DefaultProcessor) setState(state prprocessor.State) {
	pp.stateLock.Lock()
	defer pp.stateLock.Unlock()

	if state <= pp.state {
		return
	}

	pp.state = state
}

func (pp *DefaultProcessor) Proposal() ballot.Proposal {
	return pp.proposal
}

func (pp *DefaultProcessor) Block() block.Block {
	pp.RLock()
	defer pp.RUnlock()

	return pp.blk
}

func (pp *DefaultProcessor) Statics() map[string]interface{} {
	return nil
}

func (pp *DefaultProcessor) SetACCEPTVoteproof(acceptVoteproof base.Voteproof) error {
	pp.Lock()
	defer pp.Unlock()

	if pp.blk == nil {
		return xerrors.Errorf("empty block, not prepared")
	}

	if m := acceptVoteproof.Majority(); m == nil {
		return xerrors.Errorf("acceptVoteproof has empty majority")
	} else if fact, ok := m.(ballot.ACCEPTBallotFact); !ok {
		return xerrors.Errorf("acceptVoteproof does not have ballot.ACCEPTBallotFact")
	} else if !pp.blk.Hash().Equal(fact.NewBlock()) {
		return xerrors.Errorf("hash of the processed block does not match with acceptVoteproof")
	}

	pp.blk = pp.blk.SetACCEPTVoteproof(acceptVoteproof)

	return pp.blockFS.AddACCEPTVoteproof(pp.blk.Height(), pp.blk.Hash(), acceptVoteproof)
}

func (pp *DefaultProcessor) Cancel() error {
	if pp.State() == prprocessor.Canceled {
		return nil
	}

	pp.Lock()
	defer pp.Unlock()

	if err := pp.resetPrepare(); err != nil {
		return err
	}

	pp.setState(prprocessor.Canceled)

	return nil
}

func (pp *DefaultProcessor) getSuffrageInfo() (block.SuffrageInfoV0, error) {
	var ns []base.Node
	for _, address := range pp.suffrage.Nodes() {
		if address.Equal(pp.local.Address()) {
			ns = append(ns, pp.local)
		} else if n, found := pp.nodepool.Node(address); !found {
			return block.SuffrageInfoV0{}, xerrors.Errorf("suffrage node, %s not found in node pool", address)
		} else {
			ns = append(ns, n)
		}
	}

	return block.NewSuffrageInfoV0(pp.proposal.Node(), ns), nil
}
