package leveldbstorage

import (
	"context"

	"github.com/syndtr/goleveldb/leveldb"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BlockStorage struct {
	st    *Storage
	block block.Block
	batch *leveldb.Batch
}

func NewBlockStorage(st *Storage, blk block.Block) (*BlockStorage, error) {
	bst := &BlockStorage{
		st:    st,
		block: blk,
		batch: &leveldb.Batch{},
	}

	return bst, nil
}

func (bst *BlockStorage) Block() block.Block {
	return bst.block
}

func (bst *BlockStorage) SetBlock(blk block.Block) error {
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

	if b, err := marshal(bst.st.enc, blk); err != nil {
		return err
	} else {
		bst.batch.Put(leveldbBlockHashKey(blk.Hash()), b)
	}

	if b, err := marshal(bst.st.enc, blk.Manifest()); err != nil {
		return err
	} else {
		key := leveldbManifestKey(blk.Hash())
		bst.batch.Put(key, b)
	}

	if b, err := marshal(bst.st.enc, blk.Hash()); err != nil {
		return err
	} else {
		bst.batch.Put(leveldbBlockHeightKey(blk.Height()), b)
		bst.batch.Put(leveldbManifestHeightKey(blk.Height()), b)
	}

	if err := bst.setOperationsTree(blk.OperationsTree()); err != nil {
		return err
	}

	if err := bst.setStates(blk.States()); err != nil {
		return err
	}

	bst.block = blk

	return nil
}

func (bst *BlockStorage) setOperationsTree(tr tree.FixedTree) error {
	if tr.IsEmpty() {
		return nil
	}

	if b, err := marshal(bst.st.enc, tr); err != nil { // block 1st
		return err
	} else {
		bst.batch.Put(leveldbBlockOperationsKey(bst.block), b)
	}

	// store operation hashes
	if err := tr.Traverse(func(_ int, key, _, _ []byte) (bool, error) {
		bst.batch.Put(leveldbOperationFactHashKey(valuehash.NewBytes(key)), nil)

		return true, nil
	}); err != nil {
		return err
	}

	return nil
}

func (bst *BlockStorage) setStates(sts []state.State) error {
	for i := range sts {
		if b, err := marshal(bst.st.enc, sts[i]); err != nil {
			return err
		} else {
			bst.batch.Put(leveldbStateKey(sts[i].Key()), b)
		}
	}

	return nil
}

func (bst *BlockStorage) UnstageOperationSeals(hs []valuehash.Hash) error {
	return leveldbUnstageOperationSeals(bst.st, bst.batch, hs)
}

func (bst *BlockStorage) Commit(context.Context) error {
	if bst.batch.Len() < 1 {
		if err := bst.SetBlock(bst.block); err != nil {
			return err
		}
	}

	return wrapError(bst.st.db.Write(bst.batch, nil))
}

func (bst *BlockStorage) Cancel() error {
	return nil
}

func (bst *BlockStorage) Close() error {
	return nil
}

func (bst *BlockStorage) States() map[string]interface{} {
	return nil
}
