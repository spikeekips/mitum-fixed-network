package mongodbstorage

import (
	"context"
	"sync"

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
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/logging"
)

const (
	defaultColNameInfo          = "info"
	defaultColNameBlock         = "block"
	defaultColNameManifest      = "manifest"
	defaultColNameVoteproof     = "voteproof"
	defaultColNameSeal          = "seal"
	defaultColNameOperation     = "operation"
	defaultColNameOperationSeal = "operation_seal"
	defaultColNameProposal      = "proposal"
	defaultColNameState         = "state"
)

type Storage struct {
	sync.RWMutex
	*logging.Logging
	client         *Client
	encs           *encoder.Encoders
	enc            encoder.Encoder
	confirmedBlock base.Height
}

func NewStorage(client *Client, encs *encoder.Encoders, enc encoder.Encoder) (*Storage, error) {
	// NOTE call Initialize() later.

	st := &Storage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "mongodb-storage")
		}),
		client: client,
		encs:   encs,
		enc:    enc,
	}

	if cb, err := st.loadConfirmedBlock(); err != nil {
		if !xerrors.Is(err, storage.NotFoundError) {
			return nil, err
		}
	} else {
		st.confirmedBlock = cb
	}

	return st, nil
}

func (st *Storage) loadConfirmedBlock() (base.Height, error) {
	var height base.Height
	if err := st.client.GetByID(defaultColNameInfo, confirmedBlockDocID,
		func(res *mongo.SingleResult) error {
			if i, err := loadConfirmedBlock(res.Decode, st.encs); err != nil {
				return err
			} else {
				height = i
			}

			return nil
		},
	); err != nil {
		return base.Height(0), err
	}

	return height, nil
}

func (st *Storage) ConfirmedBlock() base.Height {
	st.RLock()
	defer st.RUnlock()

	return st.confirmedBlock
}

func (st *Storage) SetConfirmedBlock(height base.Height) {
	st.Lock()
	defer st.Unlock()

	if height <= st.confirmedBlock {
		return
	}

	st.confirmedBlock = height
}

func (st *Storage) SyncerStorage() (storage.SyncerStorage, error) {
	return NewSyncerStorage(st)
}

func (st *Storage) Client() *Client {
	return st.client
}

func (st *Storage) Close() error {
	return st.client.Close()
}

func (st *Storage) Encoder() encoder.Encoder {
	return st.enc
}

func (st *Storage) Encoders() *encoder.Encoders {
	return st.encs
}

func (st *Storage) LastBlock() (block.Block, error) {
	var blk block.Block

	if err := st.client.Find(
		defaultColNameBlock,
		bson.D{{"height", bson.D{{"$lte", st.ConfirmedBlock()}}}},
		func(cursor *mongo.Cursor) (bool, error) {
			if i, err := loadBlockFromDecoder(cursor.Decode, st.encs); err != nil {
				return false, err
			} else {
				blk = i
			}

			return false, nil
		},
		options.Find().SetSort(NewFilter("height", -1).D()).SetLimit(1),
	); err != nil {
		return nil, err
	}

	return blk, nil
}

func (st *Storage) blockByFilter(filter bson.D) (block.Block, error) {
	var blk block.Block

	filter = append(filter, bson.E{"height", bson.D{{"$lte", st.ConfirmedBlock()}}})
	if err := st.client.GetByFilter(
		defaultColNameBlock,
		filter,
		func(res *mongo.SingleResult) error {
			if i, err := loadBlockFromDecoder(res.Decode, st.encs); err != nil {
				return err
			} else {
				blk = i
			}

			return nil
		},
		options.FindOne().SetSort(NewFilter("height", -1).D()),
	); err != nil {
		return nil, err
	}

	return blk, nil
}

func (st *Storage) Block(h valuehash.Hash) (block.Block, error) {
	return st.blockByFilter(NewFilter("_id", h.String()).D())
}

func (st *Storage) BlockByHeight(height base.Height) (block.Block, error) {
	return st.blockByFilter(NewFilter("height", height).D())
}

func (st *Storage) manifestByFilter(filter bson.D) (block.Manifest, error) {
	var manifest block.Manifest

	filter = append(filter, bson.E{"height", bson.D{{"$lte", st.ConfirmedBlock()}}})
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
		options.FindOne().SetSort(NewFilter("height", -1).D()),
	); err != nil {
		return nil, err
	}

	return manifest, nil
}

func (st *Storage) Manifest(h valuehash.Hash) (block.Manifest, error) {
	return st.manifestByFilter(NewFilter("_id", h.String()).D())
}

func (st *Storage) ManifestByHeight(height base.Height) (block.Manifest, error) {
	return st.manifestByFilter(NewFilter("height", height).D())
}

func (st *Storage) filterVoteproof(filter bson.D, opts ...*options.FindOptions) (base.Voteproof, error) {
	var voteproof base.Voteproof

	if err := st.client.Find(
		defaultColNameVoteproof,
		filter,
		func(cursor *mongo.Cursor) (bool, error) {
			if i, err := loadVoteproofFromDecoder(cursor.Decode, st.encs); err != nil {
				return false, err
			} else {
				voteproof = i
			}

			return false, nil
		},
		opts...,
	); err != nil {
		return nil, err
	}

	return voteproof, nil
}

func (st *Storage) LastINITVoteproof() (base.Voteproof, error) {
	return st.filterVoteproof(
		NewFilter("stage", base.StageINIT).D(),
		options.Find().SetSort(NewFilter("height", -1).D()),
	)
}

func (st *Storage) NewINITVoteproof(voteproof base.Voteproof) error {
	if doc, err := NewVoteproofDoc(voteproof, st.enc); err != nil {
		return err
	} else if _, err := st.client.Set(defaultColNameVoteproof, doc); err != nil {
		return err
	}

	return nil
}

func (st *Storage) LastINITVoteproofOfHeight(height base.Height) (base.Voteproof, error) {
	return st.filterVoteproof(
		NewFilter("height", height).Add("stage", base.StageINIT).D(),
		nil,
	)
}

func (st *Storage) LastACCEPTVoteproofOfHeight(height base.Height) (base.Voteproof, error) {
	return st.filterVoteproof(
		NewFilter("height", height).Add("stage", base.StageACCEPT).D(),
		nil,
	)
}

func (st *Storage) LastACCEPTVoteproof() (base.Voteproof, error) {
	return st.filterVoteproof(
		NewFilter("stage", base.StageACCEPT).D(),
		options.Find().SetSort(NewFilter("height", -1).D()),
	)
}

func (st *Storage) NewACCEPTVoteproof(voteproof base.Voteproof) error {
	if doc, err := NewVoteproofDoc(voteproof, st.enc); err != nil {
		return err
	} else if _, err := st.client.Set(defaultColNameVoteproof, doc); err != nil {
		return err
	}

	return nil
}

func (st *Storage) Voteproofs(callback func(base.Voteproof) (bool, error), sort bool) error {
	var dir int
	if sort {
		dir = 1
	} else {
		dir = -1
	}

	opt := options.Find()
	opt.SetSort(NewFilter("height", dir).D())

	return st.client.Find(
		defaultColNameVoteproof,
		bson.D{},
		func(cursor *mongo.Cursor) (bool, error) {
			if i, err := loadVoteproofFromDecoder(cursor.Decode, st.encs); err != nil {
				return false, err
			} else {
				return callback(i)
			}
		},
		opt,
	)
}

func (st *Storage) Seal(h valuehash.Hash) (seal.Seal, error) {
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
		return nil, err
	}

	return sl, nil
}

func (st *Storage) NewSeals(seals []seal.Seal) error {
	var models []mongo.WriteModel
	var operationModels []mongo.WriteModel

	inserted := map[valuehash.Hash]struct{}{}
	for _, sl := range seals {
		if _, found := inserted[sl.Hash()]; found {
			continue
		} else {
			inserted[sl.Hash()] = struct{}{}
		}

		doc, err := NewSealDoc(sl, st.enc)
		if err != nil {
			return err
		}

		models = append(models,
			mongo.NewDeleteOneModel().SetFilter(NewFilter("_id", doc.ID()).D()),
			mongo.NewInsertOneModel().SetDocument(doc),
		)

		if _, ok := sl.(operation.Seal); !ok {
			continue
		}

		operationModels = append(operationModels,
			mongo.NewDeleteOneModel().SetFilter(NewFilter("_id", doc.ID()).D()),
			mongo.NewInsertOneModel().SetDocument(doc),
		)
	}

	if err := st.client.Bulk(defaultColNameSeal, models); err != nil {
		return err
	}

	if len(operationModels) < 1 {
		return nil
	}

	return st.client.Bulk(defaultColNameOperationSeal, operationModels)
}

func (st *Storage) Seals(callback func(valuehash.Hash, seal.Seal) (bool, error), sort, load bool) error {
	var dir int
	if sort {
		dir = 1
	} else {
		dir = -1
	}

	opt := options.Find()
	opt.SetSort(NewFilter("hash", dir).D())

	return st.client.Find(
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

func (st *Storage) StagedOperationSeals(callback func(operation.Seal) (bool, error), sort bool) error {
	var dir int
	if sort {
		dir = 1
	} else {
		dir = -1
	}

	opt := options.Find()
	opt.SetSort(NewFilter("inserted_at", dir).D())

	return st.client.Find(
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
	var models []mongo.WriteModel

	for _, h := range seals {
		models = append(models,
			mongo.NewDeleteOneModel().SetFilter(NewFilter("_id", h.String()).D()),
		)
	}

	return st.client.Bulk(defaultColNameOperationSeal, models)
}

func (st *Storage) Proposals(callback func(ballot.Proposal) (bool, error), sort bool) error {
	var dir int
	if sort {
		dir = 1
	} else {
		dir = -1
	}

	opt := options.Find()
	opt.SetSort(NewFilter("height", dir).D())

	return st.client.Find(
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
	if doc, err := NewProposalDoc(proposal, st.enc); err != nil {
		return err
	} else if _, err := st.client.Set(defaultColNameProposal, doc); err != nil {
		return err
	}

	// TODO proposal is saved in 2 collections for performance reason.
	return st.NewSeals([]seal.Seal{proposal})
}

func (st *Storage) Proposal(height base.Height, round base.Round) (ballot.Proposal, error) {
	var proposal ballot.Proposal

	if err := st.client.Find(
		defaultColNameProposal,
		NewFilter("height", height).Add("round", round).D(),
		func(cursor *mongo.Cursor) (bool, error) {
			if i, err := loadProposalFromDecoder(cursor.Decode, st.encs); err != nil {
				return false, err
			} else {
				proposal = i
			}

			return false, nil
		},
		options.Find().SetSort(NewFilter("height", -1).Add("round", -1).D()),
	); err != nil {
		return nil, err
	}

	if proposal == nil {
		return nil, storage.NotFoundError.Errorf("proposal not found; height=%v round=%v", height, round)
	}

	return proposal, nil
}

func (st *Storage) State(key string) (state.State, bool, error) {
	var sta state.State

	if err := st.client.Find(
		defaultColNameState,
		bson.D{{"key", key}, {"height", bson.D{{"$lte", st.ConfirmedBlock()}}}},
		func(cursor *mongo.Cursor) (bool, error) {
			if i, err := loadStateFromDecoder(cursor.Decode, st.encs); err != nil {
				return false, err
			} else {
				sta = i
			}

			return false, nil
		},
		options.Find().SetSort(NewFilter("height", -1).D()).SetLimit(1),
	); err != nil {
		if xerrors.Is(err, storage.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return sta, sta != nil, nil
}

func (st *Storage) NewState(sta state.State) error {
	if doc, err := NewStateDoc(sta, st.enc); err != nil {
		return err
	} else if _, err := st.client.Set(defaultColNameState, doc); err != nil {
		return err
	}

	return nil
}

func (st *Storage) HasOperation(h valuehash.Hash) (bool, error) {
	count, err := st.client.Count(
		defaultColNameOperation,
		bson.D{
			{"hash_string", h.String()},
			{"height", bson.D{{"$lte", st.ConfirmedBlock()}}},
		},
		options.Count().SetLimit(1),
	)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (st *Storage) OpenBlockStorage(blk block.Block) (storage.BlockStorage, error) {
	return NewBlockStorage(st, blk)
}

func (st *Storage) Initialize() error {
	// TODO drop the index, which has same name
	for col, models := range defaultIndexes {
		iv := st.client.Collection(col).Indexes()

		cursor, err := iv.List(context.TODO())
		if err != nil {
			return err
		}

		var results []bson.M
		if err = cursor.All(context.TODO(), &results); err != nil {
			return err
		}

		if len(results) > 0 {
			if _, err := iv.DropAll(context.TODO()); err != nil {
				return storage.WrapError(err)
			}
		}

		if len(models) < 1 {
			continue
		}

		if _, err := iv.CreateMany(context.TODO(), models); err != nil {
			return storage.WrapError(err)
		}
	}

	return nil
}

func (st *Storage) CleanupIncompleteData() error {
	// block
	if _, err := st.client.Delete(
		defaultColNameBlock,
		bson.D{{"height", bson.D{{"$gt", st.ConfirmedBlock()}}}},
	); err != nil {
		return err
	}

	// manifest
	if _, err := st.client.Delete(
		defaultColNameManifest,
		bson.D{{"height", bson.D{{"$gt", st.ConfirmedBlock()}}}},
	); err != nil {
		return err
	}

	// operation
	if _, err := st.client.Delete(
		defaultColNameOperation,
		bson.D{{"height", bson.D{{"$gt", st.ConfirmedBlock()}}}},
	); err != nil {
		return err
	}

	// state
	if _, err := st.client.Delete(
		defaultColNameState,
		bson.D{{"height", bson.D{{"$gt", st.ConfirmedBlock()}}}},
	); err != nil {
		return err
	}

	return nil
}
