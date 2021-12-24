package blockdata

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

// Clean makes Database and Blockdata to be empty. If 'remove' is true, remove
// the Blockdata directory itself.
func Clean(db storage.Database, blockdata Blockdata, remove bool) error {
	if err := db.Clean(); err != nil {
		return err
	} else if err := blockdata.Clean(remove); err != nil {
		return err
	} else {
		return nil
	}
}

func CleanByHeight(db storage.Database, blockdata Blockdata, height base.Height) error {
	if err := db.LocalBlockdataMapsByHeight(height, func(bd block.BlockdataMap) (bool, error) {
		if err := blockdata.RemoveAll(bd.Height()); err != nil {
			if errors.Is(err, util.NotFoundError) {
				return true, nil
			}

			return false, err
		}
		return true, nil
	}); err != nil {
		return err
	}

	return db.CleanByHeight(height)
}

func CheckBlock(db storage.Database, blockdata Blockdata, networkID base.NetworkID) (block.Manifest, error) {
	m, err := storage.CheckBlock(db, networkID)
	if err != nil {
		return m, err
	}

	if found, err := blockdata.Exists(m.Height()); err != nil {
		return m, err
	} else if !found {
		return m, util.NotFoundError.Errorf("block, %d not found in block data", m.Height())
	}

	return m, nil
}
