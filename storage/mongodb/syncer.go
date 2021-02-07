package mongodbstorage

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type SyncerStorage struct {
	sync.RWMutex
	*logging.Logging
	main            *Storage
	manifestStorage *Storage
	blockStorage    *Storage
	heightFrom      base.Height
	heightTo        base.Height
}

func NewSyncerStorage(main *Storage) (*SyncerStorage, error) {
	var manifestStorage, blockStorage *Storage

	if s, err := newTempStorage(main, "manifest"); err != nil {
		return nil, err
	} else if err := s.CreateIndex(ColNameManifest, manifestIndexModels, indexPrefix); err != nil {
		return nil, err
	} else {
		manifestStorage = s
	}
	if s, err := newTempStorage(main, "block"); err != nil {
		return nil, err
	} else {
		blockStorage = s
	}

	return &SyncerStorage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "mongodb-syncer-storage")
		}),
		main:            main,
		manifestStorage: manifestStorage,
		blockStorage:    blockStorage,
		heightFrom:      base.NilHeight,
		heightTo:        base.NilHeight,
	}, nil
}

func (st *SyncerStorage) Manifest(height base.Height) (block.Manifest, bool, error) {
	return st.manifestStorage.ManifestByHeight(height)
}

func (st *SyncerStorage) Manifests(heights []base.Height) ([]block.Manifest, error) {
	var bs []block.Manifest
	for i := range heights {
		if b, found, err := st.manifestStorage.ManifestByHeight(heights[i]); !found {
			return nil, storage.NotFoundError.Errorf("manifest not found")
		} else if err != nil {
			return nil, err
		} else {
			bs = append(bs, b)
		}
	}

	return bs, nil
}

func (st *SyncerStorage) SetManifests(manifests []block.Manifest) error {
	st.Lock()
	defer st.Unlock()

	var lastManifest block.Manifest
	for _, m := range manifests {
		if lastManifest == nil {
			lastManifest = m
		} else if m.Height() > lastManifest.Height() {
			lastManifest = m
		}
	}

	var models []mongo.WriteModel
	for i := range manifests {
		m := manifests[i]
		if doc, err := NewManifestDoc(m, st.blockStorage.Encoder()); err != nil {
			return err
		} else {
			models = append(models, mongo.NewInsertOneModel().SetDocument(doc))
		}

		if h := m.Height(); st.heightFrom <= base.NilHeight || h < st.heightFrom {
			st.heightFrom = h
		}

		if h := m.Height(); h > st.heightTo {
			st.heightTo = h
		}
	}

	if err := st.manifestStorage.Client().Bulk(context.Background(), ColNameManifest, models, true); err != nil {
		return err
	}

	st.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var heights []base.Height
		for i := range manifests {
			heights = append(heights, manifests[i].Height())
		}

		return e.Interface("heights", heights)
	}).
		Hinted("from_height", st.heightFrom).
		Hinted("to_height", st.heightTo).
		Int("manifests", len(manifests)).
		Msg("set manifests")

	return st.manifestStorage.setLastBlock(lastManifest, false, false)
}

func (st *SyncerStorage) HasBlock(height base.Height) (bool, error) {
	return st.blockStorage.client.Exists(ColNameManifest, util.NewBSONFilter("height", height).D())
}

func (st *SyncerStorage) SetBlocks(blocks []block.Block) error {
	st.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var heights []base.Height
		for i := range blocks {
			heights = append(heights, blocks[i].Height())
		}

		return e.Interface("heights", heights)
	}).
		Int("blocks", len(blocks)).
		Msg("set blocks")

	var lastBlock block.Block
	for i := range blocks {
		blk := blocks[i]

		if err := st.setBlock(blk); err != nil {
			return err
		}

		if lastBlock == nil {
			lastBlock = blk
		} else if blk.Height() > lastBlock.Height() {
			lastBlock = blk
		}
	}

	return st.blockStorage.setLastBlock(lastBlock, true, false)
}

func (st *SyncerStorage) setBlock(blk block.Block) error {
	var bs storage.BlockStorage
	if st, err := st.blockStorage.OpenBlockStorage(blk); err != nil {
		return err
	} else {
		bs = st
	}

	defer func() {
		_ = bs.Close()
	}()

	if err := bs.SetBlock(context.Background(), blk); err != nil {
		return err
	} else if err := bs.Commit(context.Background()); err != nil {
		return err
	}

	return nil
}

func (st *SyncerStorage) Commit() error {
	l := st.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("from_height", st.heightFrom).
			Hinted("to_height", st.heightTo)
	})

	l.Debug().Msg("trying to commit blocks to main storage")

	var last block.Manifest
	if m, found, err := st.blockStorage.LastManifest(); err != nil || !found {
		return xerrors.Errorf("failed to get last manifest fromm storage: %w", err)
	} else {
		last = m
	}

	for _, col := range []string{
		ColNameManifest,
		ColNameSeal,
		ColNameOperation,
		ColNameOperationSeal,
		ColNameProposal,
		ColNameState,
	} {
		if err := moveWithinCol(st.blockStorage, col, st.main, col, bson.D{}); err != nil {
			l.Error().Err(err).Str("collection", col).Msg("failed to move collection")

			return err
		}
		l.Debug().Str("collection", col).Msg("moved collection")
	}

	return st.main.setLastBlock(last, false, false)
}

func (st *SyncerStorage) Close() error {
	// NOTE drop tmp database
	if err := st.manifestStorage.client.DropDatabase(); err != nil {
		return err
	}

	if err := st.blockStorage.client.DropDatabase(); err != nil {
		return err
	}

	return nil
}

func newTempStorage(main *Storage, prefix string) (*Storage, error) {
	// NOTE create new mongodb database with prefix
	var tmpClient *Client
	if c, err := main.client.New(fmt.Sprintf("sync-%s_%s", prefix, util.UUID().String())); err != nil {
		return nil, err
	} else {
		tmpClient = c
	}

	return NewStorage(tmpClient, main.Encoders(), main.Encoder(), main.Cache())
}

func moveWithinCol(from *Storage, fromCol string, to *Storage, toCol string, filter bson.D) error {
	var limit int = 100
	var models []mongo.WriteModel
	err := from.Client().Find(context.Background(), fromCol, filter, func(cursor *mongo.Cursor) (bool, error) {
		if len(models) == limit {
			if err := to.Client().Bulk(context.Background(), toCol, models, false); err != nil {
				return false, err
			} else {
				models = nil
			}
		}

		raw := util.CopyBytes(cursor.Current)
		models = append(models, mongo.NewInsertOneModel().SetDocument(bson.Raw(raw)))

		return true, nil
	})
	if err != nil {
		return err
	}

	if len(models) > 0 {
		if err := to.Client().Bulk(context.Background(), toCol, models, false); err != nil {
			return err
		}
	}

	return nil
}
