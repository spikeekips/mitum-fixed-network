package mongodbstorage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BlockStorage struct {
	st                  *Storage
	ost                 *Storage
	block               block.Block
	operations          *tree.AVLTree
	states              *tree.AVLTree
	blockModels         []mongo.WriteModel
	manifestModels      []mongo.WriteModel
	operationSealModels []mongo.WriteModel
	operationModels     []mongo.WriteModel
	stateModels         []mongo.WriteModel
	statesValue         *sync.Map
}

func NewBlockStorage(st *Storage, blk block.Block) (*BlockStorage, error) {
	var nst *Storage
	if n, err := st.New(); err != nil {
		return nil, err
	} else {
		nst = n
	}

	bst := &BlockStorage{
		st:          nst,
		ost:         st,
		block:       blk,
		statesValue: &sync.Map{},
	}

	return bst, nil
}

func (bst *BlockStorage) Block() block.Block {
	return bst.block
}

func (bst *BlockStorage) SetBlock(blk block.Block) error {
	startedf := time.Now()
	defer func() {
		bst.statesValue.Store("set-block", time.Since(startedf))
	}()

	if bst.block.Height() != blk.Height() {
		return xerrors.Errorf(
			"block has different height from initial block; initial=%d != block=%d",
			bst.block.Height(),
			blk.Height(),
		)
	}

	if bst.block.Round() != blk.Round() {
		return xerrors.Errorf(
			"block has different round from initial block; initial=%d != block=%d",
			bst.block.Round(),
			blk.Round(),
		)
	}

	if bst.blockModels != nil {
		return nil
	}

	enc := bst.st.enc

	started := time.Now()
	if doc, err := NewBlockDoc(blk, enc); err != nil {
		return err
	} else {
		bst.statesValue.Store("set-block-model", time.Since(started))
		bst.blockModels = append(bst.blockModels, mongo.NewInsertOneModel().SetDocument(doc))
	}

	started = time.Now()
	if doc, err := NewManifestDoc(blk.Manifest(), enc); err != nil {
		return err
	} else {
		bst.statesValue.Store("set-manifest-model", time.Since(started))
		bst.manifestModels = append(bst.manifestModels, mongo.NewInsertOneModel().SetDocument(doc))
	}

	if err := bst.setOperations(blk.Operations()); err != nil {
		return err
	}

	if err := bst.setStates(blk.States()); err != nil {
		return err
	}

	bst.block = blk

	return nil
}

func (bst *BlockStorage) UnstageOperationSeals(seals []valuehash.Hash) error {
	started := time.Now()
	defer func() {
		bst.statesValue.Store("unstage-operation-seals", time.Since(started))
	}()

	for _, h := range seals {
		bst.operationSealModels = append(bst.operationSealModels,
			mongo.NewDeleteOneModel().SetFilter(util.NewBSONFilter("_id", h.String()).D()),
		)
	}

	return nil
}

func (bst *BlockStorage) Commit(ctx context.Context) error {
	if err := bst.commit(ctx); err == nil {
		return nil
	} else {
		defer func() {
			started := time.Now()
			_ = bst.st.CleanByHeight(bst.block.Height())
			bst.statesValue.Store("clean-by-height", time.Since(started))
		}()

		var me mongo.CommandError
		if xerrors.Is(err, context.DeadlineExceeded) {
			return storage.TimeoutError.Wrap(err)
		} else if xerrors.As(err, &me) {
			if me.HasErrorLabel("NetworkError") {
				return storage.TimeoutError.Wrap(err)
			}
		}

		return err
	}
}

func (bst *BlockStorage) commit(ctx context.Context) error {
	started := time.Now()
	defer func() {
		bst.statesValue.Store("commit", time.Since(started))

		_ = bst.st.Close()
	}()

	if bst.blockModels == nil || bst.manifestModels == nil {
		if err := bst.SetBlock(bst.block); err != nil {
			return err
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if res, err := bst.writeModels(ctx, defaultColNameBlock, bst.blockModels); err != nil {
		return storage.WrapError(err)
	} else if res != nil && res.InsertedCount < 1 {
		return xerrors.Errorf("block not inserted")
	}

	if res, err := bst.writeModels(ctx, defaultColNameManifest, bst.manifestModels); err != nil {
		return storage.WrapError(err)
	} else if res != nil && res.InsertedCount < 1 {
		return xerrors.Errorf("manifest not inserted")
	}

	if res, err := bst.writeModels(ctx, defaultColNameOperation, bst.operationModels); err != nil {
		return storage.WrapError(err)
	} else if res != nil && res.InsertedCount < 1 {
		return xerrors.Errorf("operation not inserted")
	}

	if res, err := bst.writeModels(ctx, defaultColNameState, bst.stateModels); err != nil {
		return storage.WrapError(err)
	} else if res != nil && res.InsertedCount < 1 {
		return xerrors.Errorf("state not inserted")
	}

	if _, err := bst.writeModels(ctx, defaultColNameOperationSeal, bst.operationSealModels); err != nil {
		return storage.WrapError(err)
	}

	if err := bst.ost.setLastBlock(bst.block, true, false); err != nil {
		return err
	}

	go bst.insertCaches()

	return nil
}

func (bst *BlockStorage) setOperations(tr *tree.AVLTree) error {
	started := time.Now()
	defer func() {
		bst.statesValue.Store("set-operations", time.Since(started))
	}()

	if tr == nil || tr.Empty() {
		return nil
	}

	var models []mongo.WriteModel
	if err := tr.Traverse(func(node tree.Node) (bool, error) {
		op := node.Immutable().(operation.OperationAVLNode).Operation()

		doc, err := NewOperationDoc(op, bst.st.enc, bst.block.Height())
		if err != nil {
			return false, err
		}
		models = append(models, mongo.NewInsertOneModel().SetDocument(doc))

		return true, nil
	}); err != nil {
		return err
	}

	bst.operationModels = models
	bst.operations = tr

	return nil
}

func (bst *BlockStorage) setStates(tr *tree.AVLTree) error {
	started := time.Now()
	defer func() {
		bst.statesValue.Store("set-states", time.Since(started))
	}()

	if tr == nil || tr.Empty() {
		return nil
	}

	var models []mongo.WriteModel
	if err := tr.Traverse(func(node tree.Node) (bool, error) {
		st := node.Immutable().(state.StateV0AVLNode).State()

		doc, err := NewStateDoc(st, bst.st.enc)
		if err != nil {
			return false, err
		}
		models = append(models, mongo.NewInsertOneModel().SetDocument(doc))

		return true, nil
	}); err != nil {
		return err
	}

	bst.stateModels = models
	bst.states = tr

	return nil
}

func (bst *BlockStorage) writeModels(ctx context.Context, col string, models []mongo.WriteModel) (*mongo.BulkWriteResult, error) {
	started := time.Now()
	defer func() {
		bst.statesValue.Store(fmt.Sprintf("write-models-%s", col), time.Since(started))
	}()

	if len(models) < 1 {
		return nil, nil
	}

	opts := options.BulkWrite().SetOrdered(false)
	res, err := bst.st.client.Collection(col).BulkWrite(ctx, models, opts)
	if err != nil {
		return nil, storage.WrapError(err)
	}

	return res, nil
}

func (bst *BlockStorage) insertCaches() {
	if bst.operations != nil {
		_ = bst.operations.Traverse(func(node tree.Node) (bool, error) {
			op := node.Immutable().(operation.OperationAVLNode).Operation()

			_ = bst.ost.operationFactCache.Set(op.Fact().Hash().String(), struct{}{})
			return true, nil
		})
	}

	if bst.states != nil {
		_ = bst.states.Traverse(func(node tree.Node) (bool, error) {
			st := node.Immutable().(state.StateV0AVLNode).State()

			_ = bst.ost.stateCache.Set(st.Key(), st)
			return true, nil
		})
	}
}

func (bst *BlockStorage) States() map[string]interface{} {
	m := map[string]interface{}{}
	bst.statesValue.Range(func(key, value interface{}) bool {
		m[key.(string)] = value

		return true
	})

	return m
}
