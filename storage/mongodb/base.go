package mongodbstorage

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

const (
	defaultColNameInfo          = "info"
	defaultColNameManifest      = "manifest"
	defaultColNameSeal          = "seal"
	defaultColNameOperation     = "operation"
	defaultColNameOperationSeal = "operation_seal"
	defaultColNameProposal      = "proposal"
	defaultColNameState         = "state"
)

var allCollections = []string{
	defaultColNameInfo,
	defaultColNameManifest,
	defaultColNameSeal,
	defaultColNameOperation,
	defaultColNameOperationSeal,
	defaultColNameProposal,
	defaultColNameState,
}

type Storage struct {
	sync.RWMutex
	*logging.Logging
	client             *Client
	encs               *encoder.Encoders
	enc                encoder.Encoder
	lastManifest       block.Manifest
	lastManifestHeight base.Height
	stateCache         gcache.Cache
	sealCache          gcache.Cache
	operationFactCache gcache.Cache
	readonly           bool
}

func NewStorage(client *Client, encs *encoder.Encoders, enc encoder.Encoder) (*Storage, error) {
	// NOTE call Initialize() later.

	stateCache := gcache.New(100 * 100 * 100).LRU().
		Expiration(time.Hour * 10).
		Build()

	sealCache := gcache.New(100 * 100).LRU().
		Expiration(time.Hour * 1).
		Build()

	operationFactCache := gcache.New(100 * 100 * 100).LRU().
		Expiration(time.Hour * 10).
		Build()

	if enc == nil {
		if e, err := encs.Encoder(bsonenc.BSONType, ""); err != nil {
			return nil, err
		} else {
			enc = e
		}
	}

	return &Storage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "mongodb-storage")
		}),
		client:             client,
		encs:               encs,
		enc:                enc,
		lastManifestHeight: base.NilHeight,
		stateCache:         stateCache,
		sealCache:          sealCache,
		operationFactCache: operationFactCache,
	}, nil
}

func NewStorageFromURI(uri string, encs *encoder.Encoders) (*Storage, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, xerrors.Errorf("invalid storge uri: %w", err)
	}

	connectTimeout := time.Second * 2
	execTimeout := time.Second * 2
	{
		query := parsed.Query()
		if d, err := parseDurationFromQuery(query, "connectTimeout", connectTimeout); err != nil {
			return nil, err
		} else {
			connectTimeout = d
		}
		if d, err := parseDurationFromQuery(query, "execTimeout", execTimeout); err != nil {
			return nil, err
		} else {
			execTimeout = d
		}
	}

	var be encoder.Encoder
	if e, err := encs.Encoder(bsonenc.BSONType, ""); err != nil { // NOTE get latest bson encoder
		return nil, xerrors.Errorf("bson encoder needs for mongodb: %w", err)
	} else {
		be = e
	}

	if client, err := NewClient(uri, connectTimeout, execTimeout); err != nil {
		return nil, err
	} else if st, err := NewStorage(client, encs, be); err != nil {
		return nil, err
	} else {
		return st, nil
	}
}

func (st *Storage) Initialize() error {
	if st.readonly {
		st.lastManifestHeight = base.Height(int(^uint(0) >> 1))

		return nil
	}

	if err := st.loadLastBlock(); err != nil && !storage.IsNotFoundError(err) {
		return err
	}

	if err := st.cleanupIncompleteData(); err != nil {
		return err
	}

	return st.initialize()
}

func (st *Storage) loadLastBlock() error {
	var height base.Height
	if err := st.client.GetByID(defaultColNameInfo, lastManifestDocID,
		func(res *mongo.SingleResult) error {
			if i, err := loadLastManifest(res.Decode, st.encs); err != nil {
				return err
			} else {
				height = i
			}

			return nil
		},
	); err != nil {
		return err
	}

	switch m, found, err := st.manifestByFilter(util.NewBSONFilter("height", height).D()); {
	case err != nil:
		return xerrors.Errorf("failed to find last block of height, %v: %w", height, err)
	case !found:
		return storage.NotFoundError.Errorf("failed to find last block of height, %v", height)
	default:
		return st.setLastBlock(m, false, false)
	}
}

func (st *Storage) SaveLastBlock(height base.Height) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if cb, err := NewLastManifestDoc(height, st.enc); err != nil {
		return err
	} else if _, err := st.client.Set(defaultColNameInfo, cb); err != nil {
		return err
	}

	return nil
}

func (st *Storage) lastHeight() base.Height {
	st.RLock()
	defer st.RUnlock()

	return st.lastManifestHeight
}

func (st *Storage) LastManifest() (block.Manifest, bool, error) {
	if st.readonly {
		return st.manifestByFilter(bson.D{})
	}

	st.RLock()
	defer st.RUnlock()

	if st.lastManifest == nil {
		return nil, false, nil
	}

	return st.lastManifest, true, nil
}

func (st *Storage) setLastBlock(manifest block.Manifest, save, force bool) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	st.Lock()
	defer st.Unlock()

	if manifest == nil {
		if save {
			if err := st.SaveLastBlock(base.NilHeight); err != nil {
				return err
			}
		}

		st.lastManifest = nil
		st.lastManifestHeight = base.PreGenesisHeight

		return nil
	}

	if !force && manifest.Height() <= st.lastManifestHeight {
		return nil
	}

	if save {
		if err := st.SaveLastBlock(manifest.Height()); err != nil {
			return err
		}
	}

	st.Log().Debug().Hinted("block_height", manifest.Height()).Msg("new last block")

	st.lastManifest = manifest
	st.lastManifestHeight = manifest.Height()

	return nil
}

func (st *Storage) SyncerStorage() (storage.SyncerStorage, error) {
	if st.readonly {
		return nil, xerrors.Errorf("readonly mode")
	}

	st.Lock()
	defer st.Unlock()

	return NewSyncerStorage(st)
}

func (st *Storage) Client() *Client {
	return st.client
}

func (st *Storage) Close() error {
	return st.client.Close()
}

// Clean will drop the existing collections. To keep safe the another
// collections by user, drop collections instead of drop database.
func (st *Storage) Clean() error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	drop := func(c string) error {
		return st.client.Collection(c).Drop(context.Background())
	}

	for _, c := range allCollections {
		if err := drop(c); err != nil {
			return err
		}
	}

	if err := st.initialize(); err != nil {
		return err
	}

	st.Lock()
	defer st.Unlock()

	st.lastManifest = nil
	st.lastManifestHeight = base.NilHeight

	return nil
}

func (st *Storage) CleanByHeight(height base.Height) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if err := st.cleanByHeight(height); err != nil {
		return err
	}

	switch m, found, err := st.LastManifest(); {
	case err != nil:
		return err
	case !found:
		//
	case m.Height() == height-1:
		return nil
	}

	switch m, found, err := st.ManifestByHeight(height - 1); {
	case err != nil:
		return xerrors.Errorf("failed to find block of height, %v: %w", height-1, err)
	case !found:
		return storage.NotFoundError.Errorf("failed to find block of height, %v", height-1)
	default:
		st.stateCache.Purge()
		st.operationFactCache.Purge()

		return st.setLastBlock(m, true, true)
	}
}

func (st *Storage) Copy(source storage.Storage) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	var sst *Storage
	if s, ok := source.(*Storage); !ok {
		return xerrors.Errorf("only mongodbstorage.Storage can be allowed: %T", source)
	} else {
		sst = s
	}

	if cols, err := sst.Client().Collections(); err != nil {
		return err
	} else {
		for _, c := range cols {
			if err := st.Client().CopyCollection(sst.Client(), c, c); err != nil {
				return err
			}
		}
	}

	return nil
}

func (st *Storage) Encoder() encoder.Encoder {
	return st.enc
}

func (st *Storage) Encoders() *encoder.Encoders {
	return st.encs
}

func (st *Storage) manifestByFilter(filter bson.D) (block.Manifest, bool, error) {
	var manifest block.Manifest

	if err := st.client.GetByFilter(
		defaultColNameManifest,
		filter,
		func(res *mongo.SingleResult) error {
			if i, err := loadManifestFromDecoder(res.Decode, st.encs); err != nil {
				return err
			} else {
				manifest = i
			}

			return nil
		},
		options.FindOne().SetSort(util.NewBSONFilter("height", -1).D()),
	); err != nil {
		if xerrors.Is(err, storage.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	if manifest == nil {
		return nil, false, nil
	}

	return manifest, true, nil
}

func (st *Storage) Manifest(h valuehash.Hash) (block.Manifest, bool, error) {
	return st.manifestByFilter(util.NewBSONFilter("_id", h.String()).AddOp("height", st.lastHeight(), "$lte").D())
}

func (st *Storage) ManifestByHeight(height base.Height) (block.Manifest, bool, error) {
	return st.manifestByFilter(util.NewBSONFilter("height", height).AddOp("height", st.lastHeight(), "$lte").D())
}

func (st *Storage) Seal(h valuehash.Hash) (seal.Seal, bool, error) {
	if i, _ := st.sealCache.Get(h.String()); i != nil {
		return i.(seal.Seal), true, nil
	}

	var sl seal.Seal

	if err := st.client.GetByID(
		defaultColNameSeal,
		h.String(),
		func(res *mongo.SingleResult) error {
			if i, err := loadSealFromDecoder(res.Decode, st.encs); err != nil {
				return err
			} else {
				sl = i
			}

			return nil
		},
	); err != nil {
		if storage.IsNotFoundError(err) {
			return nil, false, nil
		}

		return nil, false, err
	}

	if sl == nil {
		return nil, false, nil
	}

	return sl, true, nil
}

func (st *Storage) NewSeals(seals []seal.Seal) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if len(seals) < 1 {
		return xerrors.Errorf("empty seals")
	}

	var models []mongo.WriteModel
	var operationModels []mongo.WriteModel

	var ops []seal.Seal
	inserted := map[string]struct{}{}
	for _, sl := range seals {
		if _, found := inserted[sl.Hash().String()]; found {
			continue
		} else {
			inserted[sl.Hash().String()] = struct{}{}
		}

		doc, err := NewSealDoc(sl, st.enc)
		if err != nil {
			return err
		}

		models = append(models,
			mongo.NewInsertOneModel().SetDocument(doc),
		)

		if _, ok := sl.(operation.Seal); !ok {
			continue
		}

		ops = append(ops, sl)
		operationModels = append(operationModels,
			mongo.NewInsertOneModel().SetDocument(doc),
		)
	}

	if err := st.client.Bulk(context.Background(), defaultColNameSeal, models, false); err != nil {
		return err
	}

	if len(operationModels) < 1 {
		return nil
	}

	if err := st.client.Bulk(context.Background(), defaultColNameOperationSeal, operationModels, false); err != nil {
		return err
	}

	go func() {
		for _, sl := range ops {
			_ = st.sealCache.Set(sl.Hash().String(), sl)
		}
	}()

	return nil
}

func (st *Storage) Seals(callback func(valuehash.Hash, seal.Seal) (bool, error), sort, load bool) error {
	var dir int
	if sort {
		dir = 1
	} else {
		dir = -1
	}

	opt := options.Find()
	opt.SetSort(util.NewBSONFilter("hash", dir).D())

	return st.client.Find(
		context.Background(),
		defaultColNameSeal,
		bson.D{},
		func(cursor *mongo.Cursor) (bool, error) {
			var h valuehash.Hash
			var sl seal.Seal

			if load {
				if i, err := loadSealFromDecoder(cursor.Decode, st.encs); err != nil {
					return false, err
				} else {
					h = i.Hash()
					sl = i
				}
			} else {
				if i, err := loadSealHashFromDecoder(cursor.Decode, st.encs); err != nil {
					return false, err
				} else {
					h = i
				}
			}

			return callback(h, sl)
		},
		opt,
	)
}

func (st *Storage) SealsByHash(
	hashes []valuehash.Hash,
	callback func(valuehash.Hash, seal.Seal) (bool, error),
	load bool,
) error {
	var hashStrings []string
	for _, h := range hashes {
		hashStrings = append(hashStrings, h.String())
	}

	opt := options.Find().
		SetSort(util.NewBSONFilter("hash", 1).D())

	return st.client.Find(
		context.Background(),
		defaultColNameSeal,
		bson.M{"hash_string": bson.M{"$in": hashStrings}},
		func(cursor *mongo.Cursor) (bool, error) {
			var h valuehash.Hash
			var sl seal.Seal

			if load {
				if i, err := loadSealFromDecoder(cursor.Decode, st.encs); err != nil {
					return false, err
				} else {
					h = i.Hash()
					sl = i
				}
			} else {
				if i, err := loadSealHashFromDecoder(cursor.Decode, st.encs); err != nil {
					return false, err
				} else {
					h = i
				}
			}

			return callback(h, sl)
		},
		opt,
	)
}

func (st *Storage) HasSeal(h valuehash.Hash) (bool, error) {
	return st.client.Exists(defaultColNameSeal, util.NewBSONFilter("_id", h.String()).D())
}

func (st *Storage) StagedOperationSeals(callback func(operation.Seal) (bool, error), sort bool) error {
	var dir int
	if sort {
		dir = 1
	} else {
		dir = -1
	}

	opt := options.Find()
	opt.SetSort(util.NewBSONFilter("inserted_at", dir).D())

	return st.client.Find(
		nil,
		defaultColNameOperationSeal,
		bson.D{},
		func(cursor *mongo.Cursor) (bool, error) {
			var sl operation.Seal
			if i, err := loadSealFromDecoder(cursor.Decode, st.encs); err != nil {
				return false, err
			} else if v, ok := i.(operation.Seal); !ok {
				return false, xerrors.Errorf("not operation.Seal: %T", i)
			} else {
				sl = v
			}

			return callback(sl)
		},
		opt,
	)
}

func (st *Storage) UnstagedOperationSeals(seals []valuehash.Hash) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	var models []mongo.WriteModel
	for _, h := range seals {
		models = append(models,
			mongo.NewDeleteOneModel().SetFilter(util.NewBSONFilter("_id", h.String()).D()),
		)
	}

	return st.client.Bulk(context.Background(), defaultColNameOperationSeal, models, false)
}

func (st *Storage) Proposals(callback func(ballot.Proposal) (bool, error), sort bool) error {
	var dir int
	if sort {
		dir = 1
	} else {
		dir = -1
	}

	opt := options.Find()
	opt.SetSort(util.NewBSONFilter("height", dir).D())

	return st.client.Find(
		nil,
		defaultColNameProposal,
		bson.D{},
		func(cursor *mongo.Cursor) (bool, error) {
			var proposal ballot.Proposal
			if i, err := loadProposalFromDecoder(cursor.Decode, st.encs); err != nil {
				return false, err
			} else {
				proposal = i
			}

			return callback(proposal)
		},
		opt,
	)
}

func (st *Storage) NewProposal(proposal ballot.Proposal) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if doc, err := NewProposalDoc(proposal, st.enc); err != nil {
		return err
	} else if _, err := st.client.Add(defaultColNameProposal, doc); err != nil {
		return err
	}

	// NOTE proposal is saved in 2 collections for performance reason.
	return st.NewSeals([]seal.Seal{proposal})
}

func (st *Storage) Proposal(height base.Height, round base.Round) (ballot.Proposal, bool, error) {
	var proposal ballot.Proposal

	if err := st.client.Find(
		nil,
		defaultColNameProposal,
		util.NewBSONFilter("height", height).Add("round", round).D(),
		func(cursor *mongo.Cursor) (bool, error) {
			if i, err := loadProposalFromDecoder(cursor.Decode, st.encs); err != nil {
				return false, err
			} else {
				proposal = i
			}

			return false, nil
		},
		options.Find().SetSort(util.NewBSONFilter("height", -1).Add("round", -1).D()),
	); err != nil {
		return nil, false, err
	}

	if proposal == nil {
		return nil, false, nil
	}

	return proposal, true, nil
}

func (st *Storage) State(key string) (state.State, bool, error) {
	if i, _ := st.stateCache.Get(key); i != nil {
		return i.(state.State), true, nil
	}

	var sta state.State

	if err := st.client.Find(
		nil,
		defaultColNameState,
		util.NewBSONFilter("key", key).AddOp("height", st.lastHeight(), "$lte").D(),
		func(cursor *mongo.Cursor) (bool, error) {
			if i, err := loadStateFromDecoder(cursor.Decode, st.encs); err != nil {
				return false, err
			} else {
				sta = i
			}

			return false, nil
		},
		options.Find().SetSort(util.NewBSONFilter("height", -1).D()).SetLimit(1),
	); err != nil {
		return nil, false, err
	}

	return sta, sta != nil, nil
}

func (st *Storage) NewState(sta state.State) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if doc, err := NewStateDoc(sta, st.enc); err != nil {
		return err
	} else if _, err := st.client.Add(defaultColNameState, doc); err != nil {
		return err
	}

	_ = st.stateCache.Set(sta.Key(), sta)

	return nil
}

func (st *Storage) HasOperationFact(h valuehash.Hash) (bool, error) {
	if st.operationFactCache.Has(h.String()) {
		return true, nil
	}

	count, err := st.client.Count(
		context.Background(),
		defaultColNameOperation,
		util.NewBSONFilter("fact_hash_string", h.String()).AddOp("height", st.lastHeight(), "$lte").D(),
		options.Count().SetLimit(1),
	)
	if err != nil {
		return false, err
	}

	if count > 0 {
		_ = st.operationFactCache.Set(h.String(), struct{}{})
	}

	return count > 0, nil
}

func (st *Storage) OpenBlockStorage(blk block.Block) (storage.BlockStorage, error) {
	if st.readonly {
		return nil, xerrors.Errorf("readonly mode")
	}

	return NewBlockStorage(st, blk)
}

func (st *Storage) initialize() error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	for col, models := range defaultIndexes {
		if err := st.createIndex(col, models); err != nil {
			return err
		}
	}

	return nil
}

func (st *Storage) cleanByHeight(height base.Height) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if height <= base.PreGenesisHeight+1 {
		return st.Clean()
	}

	opts := options.BulkWrite().SetOrdered(true)
	removeByHeight := mongo.NewDeleteManyModel().SetFilter(bson.M{"height": bson.M{"$gte": height}})

	for _, col := range []string{
		defaultColNameInfo,
		defaultColNameManifest,
		defaultColNameOperation,
		defaultColNameOperationSeal,
		defaultColNameProposal,
		defaultColNameState,
	} {

		res, err := st.client.Collection(col).BulkWrite(
			context.Background(),
			[]mongo.WriteModel{removeByHeight},
			opts,
		)
		if err != nil {
			return storage.WrapStorageError(err)
		}

		st.Log().Debug().Str("collection", col).Interface("result", res).Msg("clean collection by height")
	}

	return nil
}

func (st *Storage) cleanupIncompleteData() error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	return st.cleanByHeight(st.lastHeight() + 1)
}

func (st *Storage) createIndex(col string, models []mongo.IndexModel) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	st.Lock()
	defer st.Unlock()

	iv := st.client.Collection(col).Indexes()

	cursor, err := iv.List(context.TODO())
	if err != nil {
		return err
	}

	var existings []string
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		return err
	} else {
		for _, r := range results {
			name := r["name"].(string)
			if !strings.HasPrefix(name, indexPrefix) {
				continue
			}

			existings = append(existings, name)
		}
	}

	if len(existings) > 0 {
		for _, name := range existings {
			if _, err := iv.DropOne(context.TODO(), name); err != nil {
				return storage.WrapStorageError(err)
			}
		}
	}

	if len(models) < 1 {
		return nil
	}

	if _, err := iv.CreateMany(context.TODO(), models); err != nil {
		return storage.WrapStorageError(err)
	}

	return nil
}

func (st *Storage) New() (*Storage, error) {
	var client *Client
	if cl, err := st.client.New(""); err != nil {
		return nil, err
	} else {
		client = cl
	}

	st.RLock()
	defer st.RUnlock()

	return &Storage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "mongodb-storage")
		}),
		client:             client,
		encs:               st.encs,
		enc:                st.enc,
		lastManifest:       st.lastManifest,
		lastManifestHeight: st.lastManifestHeight,
	}, nil
}

func (st *Storage) SetInfo(key string, b []byte) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if doc, err := NewInfoDoc(key, b, st.enc); err != nil {
		return err
	} else if _, err := st.client.Set(defaultColNameInfo, doc); err != nil {
		return err
	} else {
		return nil
	}
}

func (st *Storage) Info(key string) ([]byte, bool, error) {
	var b []byte
	if err := st.client.GetByID(defaultColNameInfo, infoDocKey(key),
		func(res *mongo.SingleResult) error {
			if i, err := loadInfo(res.Decode, st.encs); err != nil {
				return err
			} else {
				b = i
			}

			return nil
		},
	); err != nil {
		if xerrors.Is(err, storage.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return b, b != nil, nil
}

func (st *Storage) Readonly() (*Storage, error) {
	if nst, err := st.New(); err != nil {
		return nil, err
	} else {
		nst.readonly = true

		return nst, nil
	}
}
