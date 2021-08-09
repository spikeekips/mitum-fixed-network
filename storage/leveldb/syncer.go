package leveldbstorage

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/syndtr/goleveldb/leveldb"
)

type SyncerSession struct {
	sync.RWMutex
	*logging.Logging
	main       *Database
	database   *Database
	heightFrom base.Height
	heightTo   base.Height
}

func NewSyncerSession(main *Database) *SyncerSession {
	return &SyncerSession{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "leveldb-syncer-database")
		}),
		main:       main,
		database:   NewMemDatabase(main.Encoders(), main.Encoder()),
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
	raw, err := st.database.DB().Get(st.manifestKey(height), nil)
	if err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, mergeError(err)
	}

	m, err := st.database.loadManifest(raw)
	if err != nil {
		if errors.Is(err, util.NotFoundError) {
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
			return nil, util.NotFoundError.Errorf("manifest not found by height")
		} else if err != nil {
			return nil, err
		} else {
			bs = append(bs, b)
		}
	}

	return bs, nil
}

func (st *SyncerSession) SetManifests(manifests []block.Manifest) error {
	st.Log().Debug().Func(func(e *zerolog.Event) {
		var heights []base.Height
		for i := range manifests {
			heights = append(heights, manifests[i].Height())
		}

		e.Interface("heights", heights)
	}).
		Int("manifests", len(manifests)).
		Msg("set manifests")

	batch := &leveldb.Batch{}

	for i := range manifests {
		m := manifests[i]
		if b, err := marshal(m, st.database.Encoder()); err != nil {
			return err
		} else {
			key := st.manifestKey(m.Height())
			batch.Put(key, b)
		}
	}

	return mergeError(st.database.DB().Write(batch, nil))
}

func (st *SyncerSession) HasBlock(height base.Height) (bool, error) {
	return st.database.db.Has(leveldbBlockHeightKey(height), nil)
}

func (st *SyncerSession) block(height base.Height) (block.Block, bool, error) {
	return st.database.blockByHeight(height)
}

func (st *SyncerSession) SetBlocks(blocks []block.Block, maps []block.BlockDataMap) error {
	if len(blocks) != len(maps) {
		return errors.Errorf("blocks and maps has different size, %d != %d", len(blocks), len(maps))
	} else {
		for i := range blocks {
			if err := block.CompareManifestWithMap(blocks[i], maps[i]); err != nil {
				return err
			}
		}
	}

	st.Log().Debug().Func(func(e *zerolog.Event) {
		var heights []base.Height
		for i := range blocks {
			heights = append(heights, blocks[i].Height())
		}

		e.Interface("heights", heights)
	}).
		Int("blocks", len(blocks)).
		Msg("set blocks")

	for i := range blocks {
		blk := blocks[i]

		st.checkHeight(blk.Height())

		if bs, err := st.database.NewSession(blk); err != nil {
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
		Int64("from_height", st.heightFrom.Int64()).
		Int64("to_height", st.heightTo.Int64()).
		Msg("trying to commit blocks")

	for i := st.heightFrom; i <= st.heightTo; i++ {
		var blk block.Block
		switch j, found, err := st.block(i); {
		case err != nil:
			return err
		case !found:
			return util.NotFoundError.Errorf("block not found")
		default:
			blk = j
		}

		var m block.BlockDataMap
		switch j, found, err := st.database.BlockDataMap(i); {
		case err != nil:
			return err
		case !found:
			return util.NotFoundError.Errorf("block data map not found")
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
	if bs, err := st.main.NewSession(blk); err != nil {
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
	return mergeError(st.database.DB().Close())
}
