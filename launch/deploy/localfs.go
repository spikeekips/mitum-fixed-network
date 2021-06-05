package deploy

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"golang.org/x/xerrors"
)

var DefaultTimeAfterRemoveBlockDataFiles = time.Minute * 30

type BlockDataCleaner struct {
	sync.RWMutex
	*logging.Logging
	*util.ContextDaemon
	bd          *localfs.BlockData
	removeAfter time.Duration
	interval    time.Duration
	targets     map[base.Height]time.Time
}

func NewBlockDataCleaner(bd *localfs.BlockData, removeAfter time.Duration) *BlockDataCleaner {
	bc := &BlockDataCleaner{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "blockdata-cleaner")
		}),
		bd:          bd,
		removeAfter: removeAfter,
		interval:    time.Minute,
		targets:     map[base.Height]time.Time{},
	}
	bc.ContextDaemon = util.NewContextDaemon("blockdata-cleaner", bc.start)

	return bc
}

func (bc *BlockDataCleaner) SetLogger(l logging.Logger) logging.Logger {
	_ = bc.ContextDaemon.SetLogger(l)

	return bc.Logging.SetLogger(l)
}

func (bc *BlockDataCleaner) start(ctx context.Context) error {
	if err := bc.findRemoveds(); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := bc.clean(ctx); err != nil {
				bc.Log().Error().Err(err).Msg("failed to clean")
			}
		}
	}
}

func (bc *BlockDataCleaner) RemoveAfter() time.Duration {
	return bc.removeAfter
}

func (bc *BlockDataCleaner) Add(height base.Height) error {
	bc.Lock()
	defer bc.Unlock()

	switch found, removed, err := bc.bd.ExistsReal(height); {
	case err != nil:
		return err
	case !found:
		return util.NotFoundError.Errorf("blockdata does not exist")
	case !removed:
		if err := bc.bd.Remove(height); err != nil {
			return err
		}
	}

	if _, found := bc.targets[height]; found {
		return nil
	}

	bc.targets[height] = localtime.UTCNow().Add(bc.removeAfter)

	return nil
}

func (bc *BlockDataCleaner) currentTargets() map[base.Height]time.Time {
	bc.RLock()
	defer bc.RUnlock()

	n := map[base.Height]time.Time{}
	for i := range bc.targets {
		n[i] = bc.targets[i]
	}

	return n
}

func (bc *BlockDataCleaner) clean(ctx context.Context) error {
	var limit int64 = 100
	sem := semaphore.NewWeighted(limit)
	eg, ctx := errgroup.WithContext(context.Background())

	targets := bc.currentTargets()
	var removed []base.Height
	for i := range targets {
		height := i
		switch ok, err := bc.checkTargetIsRemovable(height, targets[i]); {
		case err != nil:
			return err
		case !ok:
			continue
		default:
			removed = append(removed, height)
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		eg.Go(func() error {
			defer sem.Release(1)

			bc.Log().Debug().Int64("height", height.Int64()).Msg("blockdata removed")
			return bc.bd.RemoveAll(height)
		})
	}

	if err := sem.Acquire(ctx, limit); err != nil {
		if !xerrors.Is(err, context.Canceled) {
			return err
		}
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	bc.Lock()
	for i := range removed {
		delete(bc.targets, removed[i])
	}
	bc.Unlock()

	go func() {
		select {
		case <-ctx.Done():
		default:
			<-time.After(bc.interval)

			if err := bc.clean(ctx); err != nil {
				bc.Log().Error().Err(err).Msg("failed to clean")
			}
		}
	}()

	return nil
}

func (bc *BlockDataCleaner) checkTargetIsRemovable(height base.Height, t time.Time) (bool, error) {
	if found, _, err := bc.bd.ExistsReal(height); err != nil {
		return false, err
	} else if !found {
		return false, nil
	}

	return localtime.UTCNow().After(t), nil
}

func (bc *BlockDataCleaner) findRemoveds() error {
	var removeds []string
	switch i, err := bc.findRemovedDirectory(); {
	case err != nil:
		return err
	case len(i) < 1:
		return nil
	default:
		removeds = i
	}

	var heights []base.Height
	switch i, err := bc.loadManifestFromRemoveds(removeds); {
	case err != nil:
		return err
	case len(i) < 1:
		return nil
	default:
		heights = i
	}

	bc.Lock()
	for i := range heights {
		height := heights[i]
		if height <= base.NilHeight {
			continue
		}

		bc.targets[height] = localtime.UTCNow() // NOTE clean immediately
	}
	bc.Unlock()

	return nil
}

func (bc *BlockDataCleaner) findRemovedDirectory() ([]string, error) {
	var removeds []string
	err := filepath.WalkDir(bc.bd.Root(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if xerrors.Is(err, fs.ErrPermission) {
				return nil
			}

			return err
		}

		if !d.IsDir() {
			return nil
		}

		removed := filepath.Join(path, localfs.BlockDirectoryRemovedTouchFile)
		if _, err := os.Stat(removed); err != nil {
			if xerrors.Is(err, fs.ErrNotExist) {
				return nil
			}

			return err
		}

		removeds = append(removeds, path)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return removeds, nil
}

func (bc *BlockDataCleaner) loadManifestFromRemoveds(removeds []string) ([]base.Height, error) {
	var limit int64 = 100
	sem := semaphore.NewWeighted(limit)
	eg, ctx := errgroup.WithContext(context.Background())

	heights := make([]base.Height, len(removeds))
	for i := range removeds {
		i := i
		f := removeds[i]

		if err := sem.Acquire(ctx, 1); err != nil {
			return nil, err
		}

		eg.Go(func() error {
			defer sem.Release(1)

			if j, err := bc.loadRemoved(f); err != nil {
				heights[i] = base.NilHeight
			} else {
				heights[i] = j.Height()
			}

			return nil
		})
	}

	if err := sem.Acquire(ctx, limit); err != nil {
		if !xerrors.Is(err, context.Canceled) {
			return nil, err
		}
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return heights, nil
}

func (bc *BlockDataCleaner) loadRemoved(path string) (block.Manifest, error) {
	g := filepath.Join(path, fmt.Sprintf("*-%s-*", block.BlockDataManifest))
	var f string
	switch matches, err := filepath.Glob(g); {
	case err != nil:
		return nil, storage.WrapStorageError(err)
	case len(matches) < 1:
		return nil, util.NotFoundError.Errorf("manifest block data not found")
	default:
		f = matches[0]
	}

	var r io.Reader
	if i, err := os.Open(filepath.Clean(f)); err != nil {
		return nil, storage.WrapStorageError(err)
	} else if j, err := util.NewGzipReader(i); err != nil {
		return nil, storage.WrapStorageError(err)
	} else {
		r = j
	}

	return bc.bd.Writer().ReadManifest(r)
}
