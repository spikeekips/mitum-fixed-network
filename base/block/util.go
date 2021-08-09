package block

import "github.com/pkg/errors"

func CompareManifestWithMap(manifest Manifest, bd BlockDataMap) error {
	if manifest.Height() != bd.Height() {
		return errors.Errorf("height not matched; %d != %d", manifest.Height(), bd.Height())
	}

	if !manifest.Hash().Equal(bd.Block()) {
		return errors.Errorf("hash not matched; %q != %q", manifest.Height(), bd.Hash())
	}

	return nil
}
