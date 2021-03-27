package localfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

func LoadBlock(st *BlockData, height base.Height) (block.Block, error) { // nolint
	blk := (interface{})(block.BlockV0{}).(block.BlockUpdater)

	if r, err := LoadData(st, height, block.BlockDataManifest); err != nil {
		return nil, err
	} else if i, err := st.Writer().ReadManifest(r); err != nil {
		return nil, err
	} else {
		blk = blk.SetManifest(i)
	}

	if r, err := LoadData(st, height, block.BlockDataOperationsTree); err != nil {
		return nil, err
	} else if i, err := st.Writer().ReadOperationsTree(r); err != nil {
		return nil, err
	} else {
		blk = blk.SetOperationsTree(i)
	}

	if r, err := LoadData(st, height, block.BlockDataOperations); err != nil {
		return nil, err
	} else if i, err := st.Writer().ReadOperations(r); err != nil {
		return nil, err
	} else {
		blk = blk.SetOperations(i)
	}

	if r, err := LoadData(st, height, block.BlockDataStatesTree); err != nil {
		return nil, err
	} else if i, err := st.Writer().ReadStatesTree(r); err != nil {
		return nil, err
	} else {
		blk = blk.SetStatesTree(i)
	}

	if r, err := LoadData(st, height, block.BlockDataStates); err != nil {
		return nil, err
	} else if i, err := st.Writer().ReadStates(r); err != nil {
		return nil, err
	} else {
		blk = blk.SetStates(i)
	}

	if r, err := LoadData(st, height, block.BlockDataINITVoteproof); err != nil {
		return nil, err
	} else if i, err := st.Writer().ReadINITVoteproof(r); err != nil {
		return nil, err
	} else {
		blk = blk.SetINITVoteproof(i)
	}

	if r, err := LoadData(st, height, block.BlockDataACCEPTVoteproof); err != nil {
		return nil, err
	} else if i, err := st.Writer().ReadACCEPTVoteproof(r); err != nil {
		return nil, err
	} else {
		blk = blk.SetACCEPTVoteproof(i)
	}

	if r, err := LoadData(st, height, block.BlockDataSuffrageInfo); err != nil {
		return nil, err
	} else if i, err := st.Writer().ReadSuffrageInfo(r); err != nil {
		return nil, err
	} else {
		blk = blk.SetSuffrageInfo(i)
	}

	if r, err := LoadData(st, height, block.BlockDataProposal); err != nil {
		return nil, err
	} else if i, err := st.Writer().ReadProposal(r); err != nil {
		return nil, err
	} else {
		blk = blk.SetProposal(i)
	}

	return blk, nil
}

func LoadData(st *BlockData, height base.Height, dataType string) (io.ReadCloser, error) {
	if found, err := st.Exists(height); err != nil {
		return nil, err
	} else if !found {
		return nil, util.NotFoundError.Errorf("block data %d not found", height)
	}

	g := filepath.Join(st.heightDirectory(height, true), fmt.Sprintf(BlockFileGlobFormats, height, dataType))

	var f string
	switch matches, err := filepath.Glob(g); {
	case err != nil:
		return nil, storage.WrapStorageError(err)
	case len(matches) < 1:
		return nil, util.NotFoundError.Errorf("block data, %q(%d) not found", dataType, height)
	default:
		f = matches[0]
	}

	if i, err := os.Open(filepath.Clean(f)); err != nil {
		return nil, storage.WrapStorageError(err)
	} else if j, err := util.NewGzipReader(i); err != nil {
		return nil, storage.WrapStorageError(err)
	} else {
		return j, nil
	}
}
