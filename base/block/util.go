package block

import "golang.org/x/xerrors"

func CompareManifestWithMap(manifest Manifest, bd BlockDataMap) error {
	if manifest.Height() != bd.Height() {
		return xerrors.Errorf("height not matched; %d != %d", manifest.Height(), bd.Height())
	}

	if !manifest.Hash().Equal(bd.Block()) {
		return xerrors.Errorf("hash not matched; %q != %q", manifest.Height(), bd.Hash())
	}

	return nil
}
