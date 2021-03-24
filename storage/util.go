package storage

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"golang.org/x/xerrors"
)

func CheckBlock(st Database, networkID base.NetworkID) error {
	var m block.Manifest
	switch b, err := CheckBlockEmpty(st); {
	case err != nil:
		return err
	case b == nil:
		return NotFoundError.Errorf("empty block manifest")
	default:
		m = b
	}

	if err := m.IsValid(networkID); err != nil {
		return xerrors.Errorf("invalid block manifest found, clean up block: %w", err)
	}

	return nil
}

// CheckBlockEmpty checks whether local has block data. If empty, return nil
// block.Block.
func CheckBlockEmpty(st Database) (block.Manifest, error) {
	switch m, found, err := st.LastManifest(); {
	case err != nil:
		return nil, err
	case !found:
		return nil, nil
	default:
		return m, nil
	}
}
