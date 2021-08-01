package mongodbstorage

import (
	"context"
	"sync"
	"time"

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
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

const (
	ColNameInfo          = "info"
	ColNameManifest      = "manifest"
	ColNameSeal          = "seal"
	ColNameOperation     = "operation"
	ColNameOperationSeal = "operation_seal"
	ColNameProposal      = "proposal"
	ColNameState         = "state"
	ColNameVoteproof     = "voteproof"
	ColNameBlockDataMap  = "blockdata_map"
)

var allCollections = []string{
	ColNameInfo,
	ColNameManifest,
	ColNameSeal,
	ColNameOperation,
	ColNameOperationSeal,
	ColNameProposal,
	ColNameState,
	ColNameVoteproof,
	ColNameBlockDataMap,
}

type Database struct {
	sync.RWMutex
	*logging.Logging
	client              *Client
	encs                *encoder.Encoders
	enc                 encoder.Encoder
	lastManifest        block.Manifest
	lastManifestHeight  base.Height
	stateCache          cache.Cache
	sealCache           cache.Cache
	operationFactCache  cache.Cache
	readonly            bool
	cache               cache.Cache
	lastINITVoteproof   base.Voteproof
	lastACCEPTVoteproof base.Voteproof
}

func NewDatabase(client *Client, encs *encoder.Encoders, enc encoder.Encoder, ca cache.Cache) (*Database, error) {
	// NOTE call Initialize() later.
	if ca == nil {
		if c, err := cache.NewGCache("lru", 100*100*100, time.Minute*3); err != nil {
			return nil, err
		} else {
			ca = c
		}
	}

	var stateCache, sealCache, operationFactCache cache.Cache
	if ca, err := ca.New(); err != nil {
		return nil, err
	} else {
		stateCache = ca
	}

	if ca, err := ca.New(); err != nil {
		return nil, err
	} else {
		sealCache = ca
	}

	if ca, err := ca.New(); err != nil {
		return nil, err
	} else {
		operationFactCache = ca
	}

	if enc == nil {
		if e, err := encs.Encoder(bsonenc.BSONEncoderType, ""); err != nil {
			return nil, err
		} else {
			enc = e
		}
	}

	return &Database{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "mongodb-database")
		}),
		client:             client,
		encs:               encs,
		enc:                enc,
		lastManifestHeight: base.NilHeight,
		stateCache:         stateCache,
		sealCache:          sealCache,
		operationFactCache: operationFactCache,
		cache:              ca,
	}, nil
}

func NewDatabaseFromURI(uri string, encs *encoder.Encoders, ca cache.Cache) (*Database, error) {
	parsed, err := network.ParseURL(uri, false)
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
	if e, err := encs.Encoder(bsonenc.BSONEncoderType, ""); err != nil { // NOTE get latest bson encoder
		return nil, xerrors.Errorf("bson encoder needs for mongodb: %w", err)
	} else {
		be = e
	}

	if client, err := NewClient(uri, connectTimeout, execTimeout); err != nil {
		return nil, err
	} else if st, err := NewDatabase(client, encs, be, ca); err != nil {
		return nil, err
	} else {
		return st, nil
	}
}

func (st *Database) Initialize() error {
	if st.readonly {
		st.lastManifestHeight = base.Height(int(^uint(0) >> 1))

		return nil
	}

	if err := st.loadLastBlock(); err != nil && !xerrors.Is(err, util.NotFoundError) {
		return err
	}

	if err := st.cleanupIncompleteData(); err != nil {
		return err
	}

	return st.initialize()
}

func (st *Database) loadLastBlock() error {
	var height base.Height
	if err := st.client.GetByID(ColNameInfo, lastManifestDocID,
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
		return util.NotFoundError.Errorf("failed to find last block of height, %v", height)
	default:
		return st.setLastBlock(m, false, false)
	}
}

func (st *Database) SaveLastBlock(height base.Height) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if cb, err := NewLastManifestDoc(height, st.enc); err != nil {
		return err
	} else if _, err := st.client.Set(ColNameInfo, cb); err != nil {
		return err
	}

	return nil
}

func (st *Database) lastHeight() base.Height {
	st.RLock()
	defer st.RUnlock()

	return st.lastManifestHeight
}

func (st *Database) LastManifest() (block.Manifest, bool, error) {
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

func (st *Database) setLastManifest(manifest block.Manifest, save, force bool) error {
	st.Lock()
	defer st.Unlock()

	return st.setLastManifestInternal(manifest, save, force)
}

func (st *Database) setLastManifestInternal(manifest block.Manifest, save, force bool) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

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

	switch t := manifest.(type) {
	case block.Block:
		manifest = t.Manifest()
	}

	st.lastManifest = manifest
	st.lastManifestHeight = manifest.Height()

	return nil
}

func (st *Database) setLastBlock(manifest block.Manifest, save, force bool) error {
	st.Lock()
	defer st.Unlock()

	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	lastManifestHeight := st.lastManifestHeight
	if err := st.setLastManifestInternal(manifest, save, force); err != nil {
		return err
	}

	if manifest == nil {
		st.lastINITVoteproof = nil
		st.lastACCEPTVoteproof = nil

		return nil
	}

	if !force && manifest.Height() <= lastManifestHeight {
		return nil
	}

	var initVoteproof, acceptVoteproof base.Voteproof
	if manifest.Height() > base.PreGenesisHeight {
		if i, j, err := st.lastVoteproofs(manifest.Height()); err != nil {
			return err
		} else {
			initVoteproof = i
			acceptVoteproof = j
		}
	}

	st.lastINITVoteproof = initVoteproof
	st.lastACCEPTVoteproof = acceptVoteproof

	return nil
}

func (st *Database) NewSyncerSession() (storage.SyncerSession, error) {
	if st.readonly {
		return nil, xerrors.Errorf("readonly mode")
	}

	st.Lock()
	defer st.Unlock()

	return NewSyncerSession(st)
}

func (st *Database) Client() *Client {
	return st.client
}

func (st *Database) Close() error {
	// FUTURE return st.client.Close()
	return nil
}

// Clean will drop the existing collections. To keep safe the another
// collections by user, drop collections instead of drop database.
func (st *Database) Clean() error {
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
	st.lastINITVoteproof = nil
	st.lastACCEPTVoteproof = nil

	return nil
}

func (st *Database) CleanByHeight(height base.Height) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if err := st.cleanByHeight(height); err != nil {
		return err
	} else if height <= base.PreGenesisHeight {
		return nil
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
		return util.NotFoundError.Errorf("failed to find block of height, %v", height-1)
	default:
		_ = st.stateCache.Purge()
		_ = st.operationFactCache.Purge()

		return st.setLastBlock(m, true, true)
	}
}

func (st *Database) Copy(source storage.Database) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	var sst *Database
	if s, ok := source.(*Database); !ok {
		return xerrors.Errorf("only mongodbstorage.Database can be allowed: %T", source)
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

func (st *Database) Encoder() encoder.Encoder {
	return st.enc
}

func (st *Database) Encoders() *encoder.Encoders {
	return st.encs
}

func (st *Database) Cache() cache.Cache {
	return st.cache
}

func (st *Database) manifestByFilter(filter bson.D) (block.Manifest, bool, error) {
	var manifest block.Manifest

	if err := st.client.GetByFilter(
		ColNameManifest,
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
		if xerrors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	if manifest == nil {
		return nil, false, nil
	}

	return manifest, true, nil
}

func (st *Database) Manifest(h valuehash.Hash) (block.Manifest, bool, error) {
	switch m, found, err := st.LastManifest(); {
	case err != nil:
		return nil, false, err
	case found && m.Hash().Equal(h):
		return m, true, nil
	}

	return st.manifestByFilter(util.NewBSONFilter("_id", h.String()).AddOp("height", st.lastHeight(), "$lte").D())
}

func (st *Database) ManifestByHeight(height base.Height) (block.Manifest, bool, error) {
	switch m, found, err := st.LastManifest(); {
	case err != nil:
		return nil, false, err
	case found && m.Height() == height:
		return m, true, nil
	}

	return st.manifestByFilter(util.NewBSONFilter("height", height).AddOp("height", st.lastHeight(), "$lte").D())
}

func (st *Database) Manifests(filter bson.M, load, reverse bool, limit int64, callback func(base.Height, valuehash.Hash, block.Manifest) (bool, error)) error {
	var dir int = 1
	if reverse {
		dir = -1
	}

	opt := options.Find().
		SetSort(util.NewBSONFilter("height", dir).D()).
		SetLimit(limit)

	if !load {
		opt = opt.SetProjection(bson.M{"height": 1, "hash": 1})
	}

	return st.client.Find(
		context.Background(),
		ColNameManifest,
		filter,
		func(cursor *mongo.Cursor) (bool, error) {
			var height base.Height
			var h valuehash.Hash
			var m block.Manifest

			if !load {
				if ht, i, err := loadManifestHeightAndHash(cursor.Decode, st.encs); err != nil {
					return false, err
				} else {
					height = ht
					h = i
				}
			} else {
				if i, err := loadManifestFromDecoder(cursor.Decode, st.encs); err != nil {
					return false, err
				} else {
					height = i.Height()
					h = i.Hash()
					m = i
				}
			}

			return callback(height, h, m)
		},
		opt,
	)
}

func (st *Database) Seal(h valuehash.Hash) (seal.Seal, bool, error) {
	if i, _ := st.sealCache.Get(h.String()); i != nil {
		return i.(seal.Seal), true, nil
	}

	var sl seal.Seal

	if err := st.client.GetByID(
		ColNameSeal,
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
		if xerrors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	if sl == nil {
		return nil, false, nil
	}

	return sl, true, nil
}

func (st *Database) NewSeals(seals []seal.Seal) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if len(seals) < 1 {
		return xerrors.Errorf("empty seals")
	}

	var models []mongo.WriteModel
	var operationModels []mongo.WriteModel

	var ops []seal.Seal
	checked := map[string]struct{}{}
	for i := range seals {
		sl := seals[i]

		if _, found := checked[sl.Hash().String()]; found {
			continue
		} else {
			checked[sl.Hash().String()] = struct{}{}
		}

		doc, err := NewSealDoc(sl, st.enc)
		if err != nil {
			return err
		}

		models = append(models, mongo.NewInsertOneModel().SetDocument(doc))

		if ok, err := st.checkNewOperationSeal(sl); err != nil {
			return err
		} else if !ok {
			continue
		} else {
			ops = append(ops, sl)
			operationModels = append(operationModels, mongo.NewInsertOneModel().SetDocument(doc))
		}
	}

	if err := st.client.Bulk(context.Background(), ColNameSeal, models, false); err != nil {
		return err
	}

	if len(operationModels) > 0 {
		if err := st.client.Bulk(context.Background(), ColNameOperationSeal, operationModels, false); err != nil {
			return err
		}
	}

	go func() {
		for _, sl := range ops {
			_ = st.sealCache.Set(sl.Hash().String(), sl, 0)
		}
	}()

	return nil
}

// checkNewOperationSeal prevents the seal, which has already processed
// operations to be stored.
func (st *Database) checkNewOperationSeal(sl seal.Seal) (bool, error) {
	var osl operation.Seal
	if i, ok := sl.(operation.Seal); !ok {
		return false, nil
	} else {
		osl = i
	}

	for i := range osl.Operations() {
		op := osl.Operations()[i]
		if found, err := st.HasOperationFact(op.Fact().Hash()); err != nil {
			return false, err
		} else if !found {
			return true, nil
		}
	}

	return false, nil
}

func (st *Database) Seals(callback func(valuehash.Hash, seal.Seal) (bool, error), sort, load bool) error {
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
		ColNameSeal,
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

func (st *Database) SealsByHash(
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
		ColNameSeal,
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

func (st *Database) HasSeal(h valuehash.Hash) (bool, error) {
	return st.client.Exists(ColNameSeal, util.NewBSONFilter("_id", h.String()).D())
}

func (st *Database) StagedOperationSeals(callback func(operation.Seal) (bool, error), sort bool) error {
	var dir int
	if sort {
		dir = 1
	} else {
		dir = -1
	}

	opt := options.Find()
	opt.SetSort(util.NewBSONFilter("inserted_at", dir).D())

	return st.client.Find(
		context.TODO(),
		ColNameOperationSeal,
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

func (st *Database) UnstagedOperationSeals(seals []valuehash.Hash) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	var models []mongo.WriteModel
	for _, h := range seals {
		models = append(models,
			mongo.NewDeleteOneModel().SetFilter(util.NewBSONFilter("_id", h.String()).D()),
		)
	}

	return st.client.Bulk(context.Background(), ColNameOperationSeal, models, false)
}

func (st *Database) Proposals(callback func(ballot.Proposal) (bool, error), sort bool) error {
	var dir int
	if sort {
		dir = 1
	} else {
		dir = -1
	}

	opt := options.Find()
	opt.SetSort(util.NewBSONFilter("height", dir).D())

	return st.client.Find(
		context.TODO(),
		ColNameProposal,
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

func (st *Database) NewProposal(proposal ballot.Proposal) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if doc, err := NewProposalDoc(proposal, st.enc); err != nil {
		return err
	} else if _, err := st.client.Add(ColNameProposal, doc); err != nil {
		return err
	}

	// NOTE proposal is saved in 2 collections for performance reason.
	return st.NewSeals([]seal.Seal{proposal})
}

func (st *Database) Proposal(height base.Height, round base.Round, proposer base.Address) (ballot.Proposal, bool, error) {
	var proposal ballot.Proposal

	if err := st.client.Find(
		context.TODO(),
		ColNameProposal,
		util.NewBSONFilter("height", height).Add("round", round).Add("proposer", proposer.String()).D(),
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

func (st *Database) State(key string) (state.State, bool, error) {
	if i, _ := st.stateCache.Get(key); i != nil {
		return i.(state.State), true, nil
	}

	var sta state.State

	if err := st.client.Find(
		context.TODO(),
		ColNameState,
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

func (st *Database) NewState(sta state.State) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if doc, err := NewStateDoc(sta, st.enc); err != nil {
		return err
	} else if _, err := st.client.Add(ColNameState, doc); err != nil {
		return err
	}

	_ = st.stateCache.Set(sta.Key(), sta, 0)

	return nil
}

func (st *Database) HasOperationFact(h valuehash.Hash) (bool, error) {
	if st.operationFactCache.Has(h.String()) {
		return true, nil
	}

	count, err := st.client.Count(
		context.Background(),
		ColNameOperation,
		util.NewBSONFilter("fact_hash_string", h.String()).AddOp("height", st.lastHeight(), "$lte").D(),
		options.Count().SetLimit(1),
	)
	if err != nil {
		return false, err
	}

	if count > 0 {
		_ = st.operationFactCache.Set(h.String(), struct{}{}, 0)
	}

	return count > 0, nil
}

func (st *Database) NewSession(blk block.Block) (storage.DatabaseSession, error) {
	if st.readonly {
		return nil, xerrors.Errorf("readonly mode")
	}

	return NewDatabaseSession(st, blk)
}

func (st *Database) initialize() error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	for col, models := range defaultIndexes {
		if err := st.CreateIndex(col, models, IndexPrefix); err != nil {
			return err
		}
	}

	return nil
}

func (st *Database) cleanByHeight(height base.Height) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if height <= base.PreGenesisHeight {
		return st.Clean()
	}

	opts := options.BulkWrite().SetOrdered(true)
	removeByHeight := mongo.NewDeleteManyModel().SetFilter(bson.M{"height": bson.M{"$gte": height}})

	for _, col := range []string{
		ColNameInfo,
		ColNameManifest,
		ColNameOperation,
		ColNameOperationSeal,
		ColNameState,
		ColNameVoteproof,
		ColNameBlockDataMap,
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

func (st *Database) cleanupIncompleteData() error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	return st.cleanByHeight(st.lastHeight() + 1)
}

func (st *Database) CreateIndex(col string, models []mongo.IndexModel, prefix string) error {
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
			if !isIndexName(name, prefix) {
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

func (st *Database) New() (*Database, error) {
	var client *Client
	if cl, err := st.client.New(""); err != nil {
		return nil, err
	} else {
		client = cl
	}

	st.RLock()
	defer st.RUnlock()

	if nst, err := NewDatabase(client, st.encs, st.enc, st.cache); err != nil {
		return nil, err
	} else {
		nst.lastManifest = st.lastManifest
		nst.lastManifestHeight = st.lastManifestHeight

		return nst, nil
	}
}

func (st *Database) SetInfo(key string, b []byte) error {
	if st.readonly {
		return xerrors.Errorf("readonly mode")
	}

	if doc, err := NewInfoDoc(key, b, st.enc); err != nil {
		return err
	} else if _, err := st.client.Set(ColNameInfo, doc); err != nil {
		return err
	} else {
		return nil
	}
}

func (st *Database) Info(key string) ([]byte, bool, error) {
	var b []byte
	if err := st.client.GetByID(ColNameInfo, infoDocKey(key),
		func(res *mongo.SingleResult) error {
			if i, err := loadInfo(res.Decode, st.encs); err != nil {
				return err
			} else {
				b = i
			}

			return nil
		},
	); err != nil {
		if xerrors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return b, b != nil, nil
}

func (st *Database) Readonly() (*Database, error) {
	if nst, err := st.New(); err != nil {
		return nil, err
	} else {
		nst.readonly = true

		return nst, nil
	}
}

func (st *Database) voteproofByFilter(filter bson.D) (base.Voteproof, bool, error) {
	var voteproof base.Voteproof

	if err := st.client.GetByFilter(
		ColNameVoteproof,
		filter,
		func(res *mongo.SingleResult) error {
			if i, err := loadVoteproofFromDecoder(res.Decode, st.encs); err != nil {
				return err
			} else {
				voteproof = i
			}

			return nil
		},
		options.FindOne().SetSort(util.NewBSONFilter("height", -1).D()),
	); err != nil {
		if xerrors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	if voteproof == nil {
		return nil, false, nil
	}

	return voteproof, true, nil
}

func (st *Database) lastVoteproofs(height base.Height) (base.Voteproof, base.Voteproof, error) {
	var initVoteproof, acceptVoteproof base.Voteproof
	switch i, found, err := st.voteproofByFilter(util.NewBSONFilter("height", height).Add("stage", base.StageINIT.String()).D()); {
	case err != nil:
		return nil, nil, xerrors.Errorf("failed to find last init voteproof of height, %v: %w", height, err)
	case !found:
		return nil, nil, util.NotFoundError.Errorf("failed to find last init voteproof of height, %v", height)
	default:
		initVoteproof = i
	}

	switch i, found, err := st.voteproofByFilter(util.NewBSONFilter("height", height).Add("stage", base.StageACCEPT.String()).D()); {
	case err != nil:
		return nil, nil, xerrors.Errorf("failed to find last accept voteproof of height, %v: %w", height, err)
	case !found:
		return nil, nil, util.NotFoundError.Errorf("failed to find last accept voteproof of height, %v", height)
	default:
		acceptVoteproof = i
	}

	return initVoteproof, acceptVoteproof, nil
}

func (st *Database) LastVoteproof(stage base.Stage) base.Voteproof {
	st.RLock()
	defer st.RUnlock()

	switch stage {
	case base.StageINIT:
		return st.lastINITVoteproof
	case base.StageACCEPT:
		return st.lastACCEPTVoteproof
	default:
		return nil
	}
}

func (st *Database) Voteproof(height base.Height, stage base.Stage) (base.Voteproof, error) {
	if l := st.LastVoteproof(stage); height == l.Height() {
		return l, nil
	}

	switch i, found, err := st.voteproofByFilter(util.NewBSONFilter("height", height).Add("stage", stage.String()).D()); {
	case err != nil:
		return nil, xerrors.Errorf("something wrong to find voteproof of height, %v and stage, %v: %w", height, stage, err)
	case !found:
		return nil, nil
	default:
		return i, nil
	}
}

func (st *Database) BlockDataMap(height base.Height) (block.BlockDataMap, bool, error) {
	var bd block.BlockDataMap

	if err := st.client.GetByFilter(
		ColNameBlockDataMap,
		util.NewBSONFilter("height", height).D(),
		func(res *mongo.SingleResult) error {
			if i, err := loadBlockDataMapFromDecoder(res.Decode, st.encs); err != nil {
				return err
			} else {
				bd = i
			}

			return nil
		},
		options.FindOne().SetSort(util.NewBSONFilter("height", -1).D()),
	); err != nil {
		if xerrors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	if bd == nil {
		return nil, false, nil
	}

	return bd, true, nil
}

func (st *Database) SetBlockDataMaps(bds []block.BlockDataMap) error {
	if len(bds) < 1 {
		return xerrors.Errorf("empty BlockDataMaps")
	}

	models := make([]mongo.WriteModel, len(bds))

	for i := range bds {
		bd := bds[i]
		if doc, err := NewBlockDataMapDoc(bd, st.enc); err != nil {
			return err
		} else {
			models[i] = mongo.NewReplaceOneModel().
				SetFilter(bson.M{"height": bd.Height()}).
				SetReplacement(doc)
		}
	}

	if res, err := writeBulkModels(
		context.Background(),
		st.client,
		ColNameBlockDataMap,
		models,
		defaultLimitWriteModels,
		options.BulkWrite().SetOrdered(false),
	); err != nil {
		return err
	} else if res != nil && res.InsertedCount < 1 {
		return err
	}

	return nil
}

func (st *Database) LocalBlockDataMapsByHeight(height base.Height, callback func(block.BlockDataMap) (bool, error)) error {
	opt := options.Find().
		SetSort(util.NewBSONFilter("height", 1).D())

	return st.client.Find(
		context.Background(),
		ColNameBlockDataMap,
		bson.M{"height": bson.M{"$gte": height}, "is_local": true},
		func(cursor *mongo.Cursor) (bool, error) {
			if i, err := loadBlockDataMapFromDecoder(cursor.Decode, st.encs); err != nil {
				return false, err
			} else {
				return callback(i)
			}
		},
		opt,
	)
}
