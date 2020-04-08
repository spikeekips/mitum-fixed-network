package isaac

import (
	"sync"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
)

type LeveldbSyncerStorage struct {
	sync.RWMutex
	*logging.Logging
	main       *LeveldbStorage
	storage    *LeveldbStorage
	heightFrom Height
	heightTo   Height
}

func NewLeveldbSyncerStorage(main *LeveldbStorage) *LeveldbSyncerStorage {
	return &LeveldbSyncerStorage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "leveldb-syncer-storage")
		}),
		main:       main,
		storage:    NewMemStorage(main.Encoders(), main.Encoder()),
		heightFrom: Height(-1),
	}
}

func (st *LeveldbSyncerStorage) manifestKey(height Height) []byte {
	return util.ConcatBytesSlice(
		leveldbTmpPrefix,
		leveldbManifestHeightKey(height),
	)
}

func (st *LeveldbSyncerStorage) Manifest(height Height) (Manifest, error) {
	raw, err := st.storage.DB().Get(st.manifestKey(height), nil)
	if err != nil {
		return nil, storage.LeveldbWrapError(err)
	}

	return st.storage.loadManifest(raw)
}

func (st *LeveldbSyncerStorage) Manifests(heights []Height) ([]Manifest, error) {
	var bs []Manifest
	for i := range heights {
		if b, err := st.Manifest(heights[i]); err != nil {
			return nil, err
		} else {
			bs = append(bs, b)
		}
	}

	return bs, nil
}

func (st *LeveldbSyncerStorage) SetManifests(manifests []Manifest) error {
	st.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var heights []Height
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
		if b, err := storage.LeveldbMarshal(st.storage.Encoder(), m); err != nil {
			return err
		} else {
			key := st.manifestKey(m.Height())
			batch.Put(key, b)
		}
	}

	return storage.LeveldbWrapError(st.storage.DB().Write(batch, nil))
}

func (st *LeveldbSyncerStorage) HasBlock(height Height) (bool, error) {
	return st.storage.db.Has(leveldbBlockHeightKey(height), nil)
}

func (st *LeveldbSyncerStorage) Block(height Height) (Block, error) {
	return st.storage.BlockByHeight(height)
}

func (st *LeveldbSyncerStorage) Blocks(heights []Height) ([]Block, error) {
	var bs []Block
	for i := range heights {
		if b, err := st.storage.BlockByHeight(heights[i]); err != nil {
			return nil, err
		} else {
			bs = append(bs, b)
		}
	}

	return bs, nil
}

func (st *LeveldbSyncerStorage) SetBlocks(blocks []Block) error {
	st.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var heights []Height
		for i := range blocks {
			heights = append(heights, blocks[i].Height())
		}

		return e.Interface("heights", heights)
	}).
		Int("blocks", len(blocks)).
		Msg("set blocks")

	for i := range blocks {
		block := blocks[i]

		st.checkHeight(block.Height())

		if bs, err := st.storage.OpenBlockStorage(block); err != nil {
			return err
		} else if err := bs.SetBlock(block); err != nil {
			return err
		} else if err := bs.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func (st *LeveldbSyncerStorage) Commit() error {
	st.Log().Debug().
		Hinted("from_height", st.heightFrom).
		Hinted("to_height", st.heightTo).
		Msg("trying to commit blocks")

	for i := st.heightFrom.Int64(); i <= st.heightTo.Int64(); i++ {
		if block, err := st.Block(Height(i)); err != nil {
			return err
		} else if err := st.commitBlock(block); err != nil {
			st.Log().Error().Err(err).Int64("height", i).Msg("failed to commit block")
			return err
		}

		st.Log().Debug().Int64("height", i).Msg("committed block")
	}

	return nil
}

func (st *LeveldbSyncerStorage) commitBlock(block Block) error {
	if bs, err := st.main.OpenBlockStorage(block); err != nil {
		return err
	} else if err := bs.SetBlock(block); err != nil {
		return err
	} else if err := bs.Commit(); err != nil {
		return err
	}

	return nil
}

func (st *LeveldbSyncerStorage) checkHeight(height Height) {
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

func (st *LeveldbSyncerStorage) Close() error {
	return storage.LeveldbWrapError(st.storage.DB().Close())
}
