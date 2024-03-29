package mongodbstorage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DatabaseSession struct {
	st                     *Database
	ost                    *Database
	block                  block.Block
	operations             tree.FixedTree
	states                 []state.State
	manifestModel          mongo.WriteModel
	operationModels        []mongo.WriteModel
	stateModels            []mongo.WriteModel
	statesValue            *sync.Map
	initVoteproofsModels   mongo.WriteModel
	acceptVoteproofsModels mongo.WriteModel
}

func NewDatabaseSession(st *Database, blk block.Block) (*DatabaseSession, error) {
	var nst *Database
	if n, err := st.New(); err != nil {
		return nil, err
	} else {
		nst = n
	}

	bst := &DatabaseSession{
		st:          nst,
		ost:         st,
		block:       blk,
		statesValue: &sync.Map{},
	}

	return bst, nil
}

func (bst *DatabaseSession) Block() block.Block {
	return bst.block
}

func (bst *DatabaseSession) SetBlock(ctx context.Context, blk block.Block) error {
	if blk == nil {
		return errors.Errorf("empty block")
	}

	finished := make(chan error)
	go func() {
		finished <- bst.setBlock(blk)
	}()

	select {
	case <-ctx.Done():
		if err := bst.st.CleanByHeight(blk.Height()); err != nil {
			if !errors.Is(err, util.NotFoundError) {
				return err
			}
		}

		return ctx.Err()
	case err := <-finished:
		return err
	}
}

func (bst *DatabaseSession) SetACCEPTVoteproof(voteproof base.Voteproof) error {
	if s := voteproof.Stage(); s != base.StageACCEPT {
		return errors.Errorf("not accept voteproof, %v", s)
	}

	if doc, err := NewVoteproofDoc(voteproof, bst.st.enc); err != nil {
		return err
	} else {
		bst.acceptVoteproofsModels = mongo.NewInsertOneModel().SetDocument(doc)

		return nil
	}
}

func (bst *DatabaseSession) setBlock(blk block.Block) error {
	startedf := time.Now()
	defer func() {
		bst.statesValue.Store("set-block", time.Since(startedf))
	}()

	if bst.block.Height() != blk.Height() {
		return errors.Errorf(
			"block has different height from initial block; initial=%d != block=%d",
			bst.block.Height(),
			blk.Height(),
		)
	}

	if bst.block.Round() != blk.Round() {
		return errors.Errorf(
			"block has different round from initial block; initial=%d != block=%d",
			bst.block.Round(),
			blk.Round(),
		)
	}

	if bst.manifestModel != nil {
		return nil
	}

	enc := bst.st.enc

	started := time.Now()
	if doc, err := NewManifestDoc(blk.Manifest(), enc); err != nil {
		return err
	} else {
		bst.statesValue.Store("set-manifest-model", time.Since(started))
		bst.manifestModel = mongo.NewInsertOneModel().SetDocument(doc)
	}

	if err := bst.setOperationsTree(blk.OperationsTree()); err != nil {
		return err
	}

	if err := bst.setStates(blk.States()); err != nil {
		return err
	}

	if err := bst.setVoteproofs(blk.ConsensusInfo().INITVoteproof(), blk.ConsensusInfo().ACCEPTVoteproof()); err != nil {
		return err
	}

	bst.block = blk

	return nil
}

func (bst *DatabaseSession) Commit(ctx context.Context, bd block.BlockdataMap) error {
	err := bst.commit(ctx, bd)
	if err == nil {
		return nil
	}

	var me mongo.CommandError
	if errors.Is(err, context.DeadlineExceeded) {
		return storage.TimeoutError.Wrap(err)
	} else if errors.As(err, &me) {
		if me.HasErrorLabel("NetworkError") {
			return storage.TimeoutError.Wrap(err)
		}
	}

	return err
}

func (bst *DatabaseSession) commit(ctx context.Context, bd block.BlockdataMap) error {
	started := time.Now()
	defer func() {
		bst.statesValue.Store("commit", time.Since(started))
	}()

	if bst.manifestModel == nil {
		if err := bst.SetBlock(ctx, bst.block); err != nil {
			return err
		}
	}

	if bst.block.Height() > base.PreGenesisHeight {
		if bst.initVoteproofsModels == nil {
			return errors.Errorf("empty init voteproof")
		}

		if bst.acceptVoteproofsModels == nil {
			return errors.Errorf("empty accept voteproof")
		}
	}

	if res, err := bst.writeModels(ctx, ColNameManifest, []mongo.WriteModel{bst.manifestModel}); err != nil {
		return MergeError(err)
	} else if res != nil && res.InsertedCount < 1 {
		return errors.Errorf("manifest not inserted")
	}

	if res, err := bst.writeModels(ctx, ColNameOperation, bst.operationModels); err != nil {
		return MergeError(err)
	} else if res != nil && res.InsertedCount < 1 {
		return errors.Errorf("operation not inserted")
	}

	if res, err := bst.writeModels(ctx, ColNameState, bst.stateModels); err != nil {
		return MergeError(err)
	} else if res != nil && res.InsertedCount < 1 {
		return errors.Errorf("state not inserted")
	}

	if bst.initVoteproofsModels != nil {
		if res, err := bst.writeModels(ctx, ColNameVoteproof, []mongo.WriteModel{bst.initVoteproofsModels}); err != nil {
			return MergeError(err)
		} else if res != nil && res.InsertedCount < 1 {
			return errors.Errorf("init voteproofs not inserted")
		}
	}

	if bst.acceptVoteproofsModels != nil {
		if res, err := bst.writeModels(ctx, ColNameVoteproof, []mongo.WriteModel{bst.acceptVoteproofsModels}); err != nil {
			return MergeError(err)
		} else if res != nil && res.InsertedCount < 1 {
			return errors.Errorf("accept voteproofs not inserted")
		}
	}

	if doc, err := NewBlockdataMapDoc(bd, bst.st.enc); err != nil {
		return err
	} else if res, err := bst.writeModels(ctx, ColNameBlockdataMap, []mongo.WriteModel{mongo.NewInsertOneModel().SetDocument(doc)}); err != nil {
		return MergeError(err)
	} else if res != nil && res.InsertedCount < 1 {
		return errors.Errorf("blockdatamap not inserted")
	}

	if err := bst.ost.setLastBlock(bst.block, true, false); err != nil {
		return err
	}

	bst.insertCaches()

	return nil
}

func (bst *DatabaseSession) setOperationsTree(tr tree.FixedTree) error {
	started := time.Now()
	defer func() {
		bst.statesValue.Store("set-operations-tree", time.Since(started))
	}()

	if tr.Len() < 1 {
		return nil
	}

	var models []mongo.WriteModel
	if err := tr.Traverse(func(no tree.FixedTreeNode) (bool, error) {
		doc, err := NewOperationDoc(valuehash.NewBytes(no.Key()), bst.st.enc, bst.block.Height())
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

func (bst *DatabaseSession) setStates(sts []state.State) error {
	started := time.Now()
	defer func() {
		bst.statesValue.Store("set-states", time.Since(started))
	}()

	var models []mongo.WriteModel
	for i := range sts {
		doc, err := NewStateDoc(sts[i], bst.st.enc)
		if err != nil {
			return err
		}
		models = append(models, mongo.NewInsertOneModel().SetDocument(doc))
	}

	bst.stateModels = models
	bst.states = sts

	return nil
}

func (bst *DatabaseSession) setVoteproofs(init, accept base.Voteproof) error {
	started := time.Now()
	defer func() {
		bst.statesValue.Store("set-voteproofs", time.Since(started))
	}()

	if init != nil {
		if doc, err := NewVoteproofDoc(init, bst.st.enc); err != nil {
			return err
		} else {
			bst.initVoteproofsModels = mongo.NewInsertOneModel().SetDocument(doc)
		}
	}

	if accept != nil {
		if err := bst.SetACCEPTVoteproof(accept); err != nil {
			return err
		}
	}

	return nil
}

func (bst *DatabaseSession) writeModels(ctx context.Context, col string, models []mongo.WriteModel) (*mongo.BulkWriteResult, error) {
	started := time.Now()
	defer func() {
		bst.statesValue.Store(fmt.Sprintf("write-models-%s", col), time.Since(started))
	}()

	if len(models) < 1 {
		return nil, nil
	}

	return writeBulkModels(
		ctx,
		bst.st.client,
		col,
		models,
		defaultLimitWriteModels,
		options.BulkWrite().SetOrdered(false),
	)
}

func (bst *DatabaseSession) insertCaches() {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		if bst.operations.Len() < 1 {
			_ = bst.operations.Traverse(func(no tree.FixedTreeNode) (bool, error) {
				_ = bst.ost.operationFactCache.Set(valuehash.NewBytes(no.Key()).String(), struct{}{}, 0)

				return true, nil
			})
		}
	}()

	go func() {
		defer wg.Done()

		for i := range bst.states {
			_ = bst.ost.stateCache.Set(bst.states[i].Key(), bst.states[i], 0)
		}
	}()

	wg.Wait()
}

func (bst *DatabaseSession) Cancel() error {
	defer func() {
		_ = bst.Close()
	}()

	if bst.block == nil {
		return errors.Errorf("empty block")
	}

	return bst.st.CleanByHeight(bst.block.Height())
}

func (bst *DatabaseSession) Close() error {
	if bst.block == nil {
		return errors.Errorf("database session already closed")
	}

	bst.states = nil
	bst.manifestModel = nil
	bst.operationModels = nil
	bst.stateModels = nil
	bst.initVoteproofsModels = nil
	bst.acceptVoteproofsModels = nil

	return bst.st.Close()
}
