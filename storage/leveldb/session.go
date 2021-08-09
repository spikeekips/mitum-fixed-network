package leveldbstorage

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/syndtr/goleveldb/leveldb"
)

type DatabaseSession struct {
	st    *Database
	block block.Block
	batch *leveldb.Batch
}

func NewSession(st *Database, blk block.Block) (*DatabaseSession, error) {
	bst := &DatabaseSession{
		st:    st,
		block: blk,
		batch: &leveldb.Batch{},
	}

	return bst, nil
}

func (bst *DatabaseSession) Block() block.Block {
	return bst.block
}

func (bst *DatabaseSession) SetBlock(_ context.Context, blk block.Block) error {
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

	if b, err := marshal(blk, bst.st.enc); err != nil {
		return err
	} else {
		bst.batch.Put(leveldbBlockHashKey(blk.Hash()), b)
	}

	if b, err := marshal(blk.Manifest(), bst.st.enc); err != nil {
		return err
	} else {
		key := leveldbManifestKey(blk.Hash())
		bst.batch.Put(key, b)
	}

	if b, err := marshal(blk.Hash(), bst.st.enc); err != nil {
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

	if err := bst.setVoteproofs(blk.ConsensusInfo().INITVoteproof(), blk.ConsensusInfo().ACCEPTVoteproof()); err != nil {
		return err
	}

	bst.block = blk

	return nil
}

func (bst *DatabaseSession) setOperationsTree(tr tree.FixedTree) error {
	if tr.Len() < 1 {
		return nil
	}

	if b, err := marshal(tr, bst.st.enc); err != nil { // block 1st
		return err
	} else {
		bst.batch.Put(leveldbBlockOperationsKey(bst.block), b)
	}

	// store operation hashes
	if err := tr.Traverse(func(no tree.FixedTreeNode) (bool, error) {
		bst.batch.Put(leveldbOperationFactHashKey(valuehash.NewBytes(no.Key())), nil)

		return true, nil
	}); err != nil {
		return err
	}

	return nil
}

func (bst *DatabaseSession) setStates(sts []state.State) error {
	for i := range sts {
		if b, err := marshal(sts[i], bst.st.enc); err != nil {
			return err
		} else {
			bst.batch.Put(leveldbStateKey(sts[i].Key()), b)
		}
	}

	return nil
}

func (bst *DatabaseSession) setVoteproofs(init, accept base.Voteproof) error {
	if init != nil {
		if b, err := marshal(init, bst.st.enc); err != nil {
			return err
		} else {
			bst.batch.Put(leveldbVoteproofKey(init.Height(), base.StageINIT), b)
		}
	}

	if accept != nil {
		if err := bst.SetACCEPTVoteproof(accept); err != nil {
			return err
		}
	}

	return nil
}

func (bst *DatabaseSession) Commit(ctx context.Context, bd block.BlockDataMap) error {
	if bst.batch.Len() < 1 {
		if err := bst.SetBlock(ctx, bst.block); err != nil {
			return err
		}
	}

	if b, err := marshal(bd, bst.st.enc); err != nil {
		return err
	} else {
		bst.batch.Put(leveldbBlockDataMapKey(bd.Height()), b)
	}

	return mergeError(bst.st.db.Write(bst.batch, nil))
}

func (bst *DatabaseSession) Cancel() error {
	return nil
}

func (bst *DatabaseSession) Close() error {
	return nil
}

func (bst *DatabaseSession) SetACCEPTVoteproof(voteproof base.Voteproof) error {
	if s := voteproof.Stage(); s != base.StageACCEPT {
		return errors.Errorf("not accept voteproof, %v", s)
	}

	if b, err := marshal(voteproof, bst.st.enc); err != nil {
		return err
	} else {
		bst.batch.Put(leveldbVoteproofKey(voteproof.Height(), base.StageACCEPT), b)

		return nil
	}
}
