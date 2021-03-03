package storage

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"golang.org/x/xerrors"
)

func CheckBlock(st Storage, blockFS *BlockFS, networkID base.NetworkID) error {
	var blk block.Block
	switch b, err := CheckBlockEmpty(st, blockFS); {
	case err != nil:
		return err
	case b == nil:
		return NotFoundError.Errorf("empty block")
	default:
		blk = b
	}

	if err := blk.IsValid(networkID); err != nil {
		return xerrors.Errorf("invalid block found, clean up block: %w", err)
	}

	return nil
}

// CheckBlockEmpty checks whether local has block data in Storage and BlockFS.
// If empty, return nil block.Block. Block should exist both in Storage and
// BlockFS, if not, returns empty.
func CheckBlockEmpty(st Storage, blockFS *BlockFS) (block.Block, error) {
	var manifest block.Manifest
	switch m, found, err := st.LastManifest(); {
	case err != nil:
		return nil, err
	case !found:
		return nil, nil
	default:
		manifest = m
	}

	if blk, err := blockFS.Load(manifest.Height()); err != nil {
		if xerrors.Is(err, NotFoundError) {
			return nil, nil
		}

		return nil, err
	} else {
		return blk, nil
	}
}

// Clean makes Storage and BlockFS to be empty. If 'remove' is true, remove
// the BlockFS directory itself.
func Clean(st Storage, blockFS *BlockFS, remove bool) error {
	if err := st.Clean(); err != nil {
		return err
	} else if err := blockFS.Clean(remove); err != nil {
		return err
	} else {
		return nil
	}
}
