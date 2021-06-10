package process

import (
	"context"
	"os"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
)

var HookNameCheckBlockDataPath = "check_blockdata_path"

func HookCheckBlockDataPath(ctx context.Context) (context.Context, error) {
	var l config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	}
	conf := l.Storage().BlockData()

	if fi, err := os.Stat(conf.Path()); err != nil {
		if !os.IsNotExist(err) { // NOTE if not exist, create new
			return ctx, err
		}

		if err := os.MkdirAll(conf.Path(), localfs.DefaultDirectoryPermission); err != nil {
			return ctx, storage.WrapFSError(err)
		}
	} else if !fi.IsDir() {
		return ctx, storage.FSError.Errorf("blockdata directory, %q not directory", conf.Path())
	}

	return ctx, nil
}
