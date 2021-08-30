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

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
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
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
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

func (bc *BlockDataCleaner) SetLogging(l *logging.Logging) *logging.Logging {
	_ = bc.ContextDaemon.SetLogging(l)

	return bc.Logging.SetLogging(l)
}

func (bc *BlockDataCleaner) start(ctx context.Context) error {
	if err := bc.findRemoveds(); err != nil {
		return err
	}

	go func() {
		if err := bc.clean(ctx); err != nil {
			bc.Log().Error().Err(err).Msg("failed to clean")
		}
	}()

	<-ctx.Done()

	return ctx.Err()
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
	if err := ctx.Err(); err != nil {
		return err
	}

	wk := util.NewErrgroupWorker(ctx, 100)
	defer wk.Close()

	targets := bc.currentTargets()
	var removed []base.Height

	errch := make(chan error, 1)
	go func() {
		defer wk.Done()

		for i := range targets {
			height := i
			switch ok, err := bc.checkTargetIsRemovable(height, targets[i]); {
			case err != nil:
				errch <- err

				return
			case !ok:
				continue
			default:
				removed = append(removed, height)
			}

			if err := wk.NewJob(func(context.Context, uint64) error {
				bc.Log().Debug().Int64("height", height.Int64()).Msg("blockdata removed")

				return bc.bd.RemoveAll(height)
			}); err != nil {
				bc.Log().Error().Err(err).Int64("height", height.Int64()).Msg("failed to remove blockdata")

				break
			}
		}

		errch <- nil
	}()

	err := wk.Wait()
	if cerr := <-errch; cerr != nil {
		return cerr
	}

	if err != nil {
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
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}

			return err
		}

		if !d.IsDir() {
			return nil
		}

		removed := filepath.Join(path, localfs.BlockDirectoryRemovedTouchFile)
		if _, err := os.Stat(removed); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
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
	wk := util.NewErrgroupWorker(context.Background(), 100)
	defer wk.Close()

	heights := make([]base.Height, len(removeds))

	go func() {
		defer wk.Done()

		for i := range removeds {
			i := i
			f := removeds[i]

			if err := wk.NewJob(func(context.Context, uint64) error {
				if j, err := bc.loadRemoved(f); err != nil {
					heights[i] = base.NilHeight
				} else {
					heights[i] = j.Height()
				}

				return nil
			}); err != nil {
				return
			}
		}
	}()

	if err := wk.Wait(); err != nil {
		return nil, err
	}

	return heights, nil
}

func (bc *BlockDataCleaner) loadRemoved(path string) (block.Manifest, error) {
	g := filepath.Join(path, fmt.Sprintf("*-%s-*", block.BlockDataManifest))
	var f string
	switch matches, err := filepath.Glob(g); {
	case err != nil:
		return nil, storage.MergeStorageError(err)
	case len(matches) < 1:
		return nil, util.NotFoundError.Errorf("manifest block data not found")
	default:
		f = matches[0]
	}

	var r io.Reader
	if i, err := os.Open(filepath.Clean(f)); err != nil {
		return nil, storage.MergeStorageError(err)
	} else if j, err := util.NewGzipReader(i); err != nil {
		return nil, storage.MergeStorageError(err)
	} else {
		r = j
	}

	return bc.bd.Writer().ReadManifest(r)
}
