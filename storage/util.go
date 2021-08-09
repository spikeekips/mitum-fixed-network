package storage

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util"
)

func CheckBlock(st Database, networkID base.NetworkID) (block.Manifest, error) {
	var m block.Manifest
	switch b, err := CheckBlockEmpty(st); {
	case err != nil:
		return nil, err
	case b == nil:
		return nil, util.NotFoundError.Errorf("empty block manifest")
	default:
		m = b
	}

	if err := m.IsValid(networkID); err != nil {
		return m, errors.Wrap(err, "invalid block manifest found, clean up block")
	}

	return m, nil
}

// CheckBlockEmpty checks whether local has block data. If empty, return nil
// block.Manifest.
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
