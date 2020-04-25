package mongodbstorage

import (
	"fmt"
	"sync"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util/logging"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongodbSyncerStorage struct {
	sync.RWMutex
	*logging.Logging
	main            *MongodbStorage
	manifestStorage *MongodbStorage
	blockStorage    *MongodbStorage
	heightFrom      base.Height
	heightTo        base.Height
}

func NewMongodbSyncerStorage(main *MongodbStorage) (*MongodbSyncerStorage, error) {
	var manifestStorage, blockStorage *MongodbStorage
	if s, err := newTempStorage(main, "manifest"); err != nil {
		return nil, err
	} else {
		manifestStorage = s
	}
	if s, err := newTempStorage(main, "block"); err != nil {
		return nil, err
	} else {
		blockStorage = s
	}

	return &MongodbSyncerStorage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "mongodb-syncer-storage")
		}),
		main:            main,
		manifestStorage: manifestStorage,
		blockStorage:    blockStorage,
		heightFrom:      base.Height(-1),
	}, nil
}

func (st *MongodbSyncerStorage) Manifest(height base.Height) (block.Manifest, error) {
	return st.manifestStorage.ManifestByHeight(height)
}

func (st *MongodbSyncerStorage) Manifests(heights []base.Height) ([]block.Manifest, error) {
	var bs []block.Manifest
	for i := range heights {
		if b, err := st.manifestStorage.ManifestByHeight(heights[i]); err != nil {
			return nil, err
		} else {
			bs = append(bs, b)
		}
	}

	return bs, nil
}

func (st *MongodbSyncerStorage) SetManifests(manifests []block.Manifest) error {
	st.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var heights []base.Height
		for i := range manifests {
			heights = append(heights, manifests[i].Height())
		}

		return e.Interface("heights", heights)
	}).
		Int("manifests", len(manifests)).
		Msg("set manifests")

	var models []mongo.WriteModel
	for i := range manifests {
		if doc, err := NewManifestDoc(manifests[i], st.blockStorage.Encoder()); err != nil {
			return err
		} else {
			models = append(models,
				mongo.NewInsertOneModel().SetDocument(doc),
			)
		}
	}

	return st.manifestStorage.client.Bulk("manifest", models)
}

func (st *MongodbSyncerStorage) HasBlock(height base.Height) (bool, error) {
	return st.blockStorage.client.Exists("block", NewFilter("height", height).D())
}

func (st *MongodbSyncerStorage) Block(height base.Height) (block.Block, error) {
	return st.blockStorage.BlockByHeight(height)
}

func (st *MongodbSyncerStorage) Blocks(heights []base.Height) ([]block.Block, error) {
	var bs []block.Block
	for i := range heights {
		if b, err := st.blockStorage.BlockByHeight(heights[i]); err != nil {
			return nil, err
		} else {
			bs = append(bs, b)
		}
	}

	return bs, nil
}

func (st *MongodbSyncerStorage) SetBlocks(blocks []block.Block) error {
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

		if bs, err := st.blockStorage.OpenBlockStorage(blk); err != nil {
			return err
		} else if err := bs.SetBlock(blk); err != nil {
			return err
		} else if err := bs.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func (st *MongodbSyncerStorage) Commit() error {
	st.Log().Debug().
		Hinted("from_height", st.heightFrom).
		Hinted("to_height", st.heightTo).
		Msg("trying to commit blocks")

	for i := st.heightFrom.Int64(); i <= st.heightTo.Int64(); i++ {
		if blk, err := st.Block(base.Height(i)); err != nil {
			return err
		} else if err := st.commitBlock(blk); err != nil {
			st.Log().Error().Err(err).Int64("height", i).Msg("failed to commit block")
			return err
		}

		st.Log().Debug().Int64("height", i).Msg("committed block")
	}

	return nil
}

func (st *MongodbSyncerStorage) commitBlock(blk block.Block) error {
	if bs, err := st.main.OpenBlockStorage(blk); err != nil {
		return err
	} else if err := bs.SetBlock(blk); err != nil {
		return err
	} else if err := bs.Commit(); err != nil {
		return err
	}

	return nil
}

func (st *MongodbSyncerStorage) checkHeight(height base.Height) {
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

func (st *MongodbSyncerStorage) Close() error {
	// NOTE drop tmp database
	if err := st.manifestStorage.client.DropDatabase(); err != nil {
		return err
	}
	if err := st.blockStorage.client.DropDatabase(); err != nil {
		return err
	}

	return st.blockStorage.client.Close()
}

func newTempStorage(main *MongodbStorage, prefix string) (*MongodbStorage, error) {
	// NOTE create new mongodb client
	var tmpClient *Client
	if uri, err := NewTempURI(main.client.uri, fmt.Sprintf("sync-%s", prefix)); err != nil {
		return nil, err
	} else if c, err := NewClient(uri, time.Second*2, main.client.execTimeout); err != nil {
		return nil, err
	} else {
		tmpClient = c
	}

	return NewMongodbStorage(tmpClient, main.Encoders(), main.Encoder()), nil
}
