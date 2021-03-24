package leveldbstorage

import (
	"context"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type SyncerSession struct {
	sync.RWMutex
	*logging.Logging
	main       *Storage
	storage    *Storage
	heightFrom base.Height
	heightTo   base.Height
}

func NewSyncerSession(main *Storage) *SyncerSession {
	return &SyncerSession{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "leveldb-syncer-storage")
		}),
		main:       main,
		storage:    NewMemStorage(main.Encoders(), main.Encoder()),
		heightFrom: base.Height(-1),
	}
}

func (st *SyncerSession) manifestKey(height base.Height) []byte {
	return util.ConcatBytesSlice(
		keyPrefixTmp,
		leveldbManifestHeightKey(height),
	)
}

func (st *SyncerSession) Manifest(height base.Height) (block.Manifest, bool, error) {
	raw, err := st.storage.DB().Get(st.manifestKey(height), nil)
	if err != nil {
		if xerrors.Is(err, storage.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, wrapError(err)
	}

	m, err := st.storage.loadManifest(raw)
	if err != nil {
		if xerrors.Is(err, storage.NotFoundError) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return m, true, nil
}

func (st *SyncerSession) Manifests(heights []base.Height) ([]block.Manifest, error) {
	var bs []block.Manifest
	for i := range heights {
		if b, found, err := st.Manifest(heights[i]); !found {
			return nil, storage.NotFoundError.Errorf("manifest not found by height")
		} else if err != nil {
			return nil, err
		} else {
			bs = append(bs, b)
		}
	}

	return bs, nil
}

func (st *SyncerSession) SetManifests(manifests []block.Manifest) error {
	st.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var heights []base.Height
		for i := range manifests {
			heights = append(heights, manifests[i].Height())
		}

		return e.Interface("heights", heights)
	}).
		Int("manifests", len(manifests)).
		Msg("set manifests")

	batch := &leveldb.Batch{}

	for i := range manifests {
		m := manifests[i]
		if b, err := marshal(st.storage.Encoder(), m); err != nil {
			return err
		} else {
			key := st.manifestKey(m.Height())
			batch.Put(key, b)
		}
	}

	return wrapError(st.storage.DB().Write(batch, nil))
}

func (st *SyncerSession) HasBlock(height base.Height) (bool, error) {
	return st.storage.db.Has(leveldbBlockHeightKey(height), nil)
}

func (st *SyncerSession) block(height base.Height) (block.Block, bool, error) {
	return st.storage.blockByHeight(height)
}

func (st *SyncerSession) SetBlocks(blocks []block.Block, maps []block.BlockDataMap) error {
	if len(blocks) != len(maps) {
		return xerrors.Errorf("blocks and maps has different size, %d != %d", len(blocks), len(maps))
	} else {
		for i := range blocks {
			if err := block.CompareManifestWithMap(blocks[i], maps[i]); err != nil {
				return err
			}
		}
	}

	st.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var heights []base.Height
		for i := range blocks {
			heights = append(heights, blocks[i].Height())
		}

		return e.Interface("heights", heights)
	}).
		Int("blocks", len(blocks)).
		Msg("set blocks")

	for i := range blocks {
		blk := blocks[i]

		st.checkHeight(blk.Height())

		if bs, err := st.storage.NewStorageSession(blk); err != nil {
			return err
		} else if err := bs.SetBlock(context.Background(), blk); err != nil {
			return err
		} else if err := bs.Commit(context.Background(), maps[i]); err != nil {
			return err
		}
	}

	return nil
}

func (st *SyncerSession) Commit() error {
	st.Log().Debug().
		Hinted("from_height", st.heightFrom).
		Hinted("to_height", st.heightTo).
		Msg("trying to commit blocks")

	for i := st.heightFrom; i <= st.heightTo; i++ {
		var blk block.Block
		switch j, found, err := st.block(i); {
		case err != nil:
			return err
		case !found:
			return storage.NotFoundError.Errorf("block not found")
		default:
			blk = j
		}

		var m block.BlockDataMap
		switch j, found, err := st.storage.BlockDataMap(i); {
		case err != nil:
			return err
		case !found:
			return storage.NotFoundError.Errorf("block data map not found")
		default:
			m = j
		}

		if err := st.commitBlock(blk, m); err != nil {
			st.Log().Error().Err(err).Int64("height", i.Int64()).Msg("failed to commit block")

			return err
		}

		st.Log().Debug().Int64("height", i.Int64()).Msg("committed block")
	}

	return nil
}

func (st *SyncerSession) commitBlock(blk block.Block, m block.BlockDataMap) error {
	if bs, err := st.main.NewStorageSession(blk); err != nil {
		return err
	} else if err := bs.SetBlock(context.Background(), blk); err != nil {
		return err
	} else if err := bs.Commit(context.Background(), m); err != nil {
		return err
	}

	return nil
}

func (st *SyncerSession) checkHeight(height base.Height) {
	st.Lock()
	defer st.Unlock()

	switch {
	case st.heightFrom < 0:
		st.heightFrom = height
		st.heightTo = height
	case st.heightFrom > height:
		st.heightFrom = height
	case st.heightTo < height:
		st.heightTo = height
	}
}

func (st *SyncerSession) Close() error {
	return wrapError(st.storage.DB().Close())
}
