package mongodbstorage

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	syncTempDatabaseNamePrefix       = "sync-"
	syncTempDatabaseNamePrefixRegexp = `^sync\-`
)

type SyncerSession struct {
	sync.RWMutex
	*logging.Logging
	main             *Database
	manifestDatabase *Database
	session          *Database
	heightFrom       base.Height
	heightTo         base.Height
	skipLastBlock    bool
}

func NewSyncerSession(main *Database) (*SyncerSession, error) {
	var manifestDatabase, session *Database

	if s, err := newTempDatabase(main, "manifest"); err != nil {
		return nil, err
	} else if err := s.CreateIndex(ColNameManifest, manifestIndexModels, IndexPrefix); err != nil {
		return nil, err
	} else {
		manifestDatabase = s
	}
	if s, err := newTempDatabase(main, "block"); err != nil {
		return nil, err
	} else {
		session = s
	}

	return &SyncerSession{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "mongodb-syncer-database")
		}),
		main:             main,
		manifestDatabase: manifestDatabase,
		session:          session,
		heightFrom:       base.NilHeight,
		heightTo:         base.NilHeight,
	}, nil
}

func (st *SyncerSession) Manifest(height base.Height) (block.Manifest, bool, error) {
	return st.manifestDatabase.ManifestByHeight(height)
}

func (st *SyncerSession) SetManifests(manifests []block.Manifest) error {
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
		if doc, err := NewManifestDoc(m, st.session.Encoder()); err != nil {
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

	if err := st.manifestDatabase.Client().Bulk(context.Background(), ColNameManifest, models, true); err != nil {
		return err
	}

	st.Log().Trace().Func(func(e *zerolog.Event) {
		var heights []base.Height
		for i := range manifests {
			heights = append(heights, manifests[i].Height())
		}

		e.Interface("heights", heights)
	}).
		Int64("from_height", st.heightFrom.Int64()).
		Int64("to_height", st.heightTo.Int64()).
		Int("manifests", len(manifests)).
		Msg("set manifests")

	return st.manifestDatabase.setLastManifest(lastManifest, false, false)
}

func (st *SyncerSession) HasBlock(height base.Height) (bool, error) {
	return st.session.client.Exists(ColNameManifest, util.NewBSONFilter("height", height).D())
}

func (st *SyncerSession) SetBlocks(blocks []block.Block, maps []block.BlockdataMap) error {
	if len(blocks) != len(maps) {
		return errors.Errorf("blocks and maps has different size, %d != %d", len(blocks), len(maps))
	} else {
		for i := range blocks {
			if err := block.CompareManifestWithMap(blocks[i], maps[i]); err != nil {
				return err
			}
		}
	}

	st.Log().Trace().Func(func(e *zerolog.Event) {
		var heights []base.Height
		for i := range blocks {
			heights = append(heights, blocks[i].Height())
		}

		e.Interface("heights", heights)
	}).
		Int("blocks", len(blocks)).
		Msg("set blocks")

	var lastBlock block.Block
	for i := range blocks {
		blk := blocks[i]
		m := maps[i]

		if err := st.setBlock(blk, m); err != nil {
			return err
		}

		if lastBlock == nil {
			lastBlock = blk
		} else if blk.Height() > lastBlock.Height() {
			lastBlock = blk
		}
	}

	return st.session.setLastBlock(lastBlock, true, false)
}

func (st *SyncerSession) setBlock(blk block.Block, m block.BlockdataMap) error {
	var bs storage.DatabaseSession
	if st, err := st.session.NewSession(blk); err != nil {
		return err
	} else {
		bs = st
	}

	defer func() {
		_ = bs.Close()
	}()

	if err := bs.SetBlock(context.Background(), blk); err != nil {
		return err
	} else if err := bs.Commit(context.Background(), m); err != nil {
		return err
	}

	return nil
}

func (st *SyncerSession) Commit() error {
	l := st.Log().With().Int64("from_height", st.heightFrom.Int64()).
		Int64("to_height", st.heightTo.Int64()).
		Logger()

	var last block.Manifest
	switch m, found, err := st.session.LastManifest(); {
	case err != nil:
		err = errors.Wrap(err, "failed to get last manifest from storage")

		l.Error().Err(err).Msg("failed to commit blocks to main database")

		return err
	case !found:
		err = util.NotFoundError.Errorf("failed to get manifest from storage")

		l.Error().Err(err).Msg("failed to commit blocks to main database")

		return err
	default:
		last = m
	}

	for _, col := range []string{
		ColNameManifest,
		ColNameOperation,
		ColNameStagedOperation,
		ColNameProposal,
		ColNameState,
		ColNameVoteproof,
		ColNameBlockdataMap,
	} {
		if err := moveWithinCol(st.session, col, st.main, col, bson.D{}); err != nil {
			l.Error().Err(err).Str("collection", col).Msg("failed to move collection")

			return err
		}
		l.Trace().Str("collection", col).Msg("moved collection")
	}

	if !st.skipLastBlock {
		if err := st.main.setLastBlock(last, false, false); err != nil {
			l.Error().Err(err).Msg("failed to commit blocks to main database")

			return err
		}
	}

	l.Debug().Msg("blocks committed to main database")

	return nil
}

func (st *SyncerSession) Close() error {
	// NOTE drop tmp database
	if err := st.manifestDatabase.client.DropDatabase(); err != nil {
		return err
	}

	if err := st.session.client.DropDatabase(); err != nil {
		return err
	}

	return nil
}

func (st *SyncerSession) SetSkipLastBlock(b bool) {
	st.Lock()
	defer st.Unlock()

	st.skipLastBlock = b
}

func newTempDatabase(main *Database, prefix string) (*Database, error) {
	// NOTE create new mongodb database with prefix
	var tmpClient *Client
	if c, err := main.client.New(fmt.Sprintf("%s%s_%s", syncTempDatabaseNamePrefix, prefix, util.UUID().String())); err != nil {
		return nil, err
	} else {
		tmpClient = c
	}

	return NewDatabase(tmpClient, main.Encoders(), main.Encoder(), main.Cache())
}

func moveWithinCol(from *Database, fromCol string, to *Database, toCol string, filter bson.D) error {
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

func CleanTemporayDatabase(st *Database) error {
	dbs, err := st.client.Databases(bson.M{"name": bson.M{"$regex": syncTempDatabaseNamePrefixRegexp}})
	switch {
	case err != nil:
		return MergeError(errors.Wrap(err, "failed to get databases"))
	case len(dbs) < 1:
		return nil
	}

	for i := range dbs {
		db := dbs[i]
		if err := st.client.Database(db).Drop(context.Background()); err != nil {
			return MergeError(errors.Wrapf(err, "failed to drop database, %q", db))
		}
	}

	return nil
}
