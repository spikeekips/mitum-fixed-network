package leveldbstorage

import (
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	leveldbStorage "github.com/syndtr/goleveldb/leveldb/storage"
	leveldbutil "github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	leveldbTmpPrefix                        []byte = []byte{0x00, 0x00}
	leveldbBlockHeightPrefix                []byte = []byte{0x00, 0x01}
	leveldbBlockHashPrefix                  []byte = []byte{0x00, 0x02}
	leveldbManifestPrefix                   []byte = []byte{0x00, 0x03}
	leveldbVoteproofHeightPrefix            []byte = []byte{0x00, 0x04}
	leveldbSealPrefix                       []byte = []byte{0x00, 0x05}
	leveldbSealHashPrefix                   []byte = []byte{0x00, 0x06}
	leveldbProposalPrefix                   []byte = []byte{0x00, 0x07}
	leveldbBlockOperationsPrefix            []byte = []byte{0x00, 0x08}
	leveldbBlockStatesPrefix                []byte = []byte{0x00, 0x09}
	leveldbStagedOperationSealPrefix        []byte = []byte{0x00, 0x10}
	leveldbStagedOperationSealReversePrefix []byte = []byte{0x00, 0x11}
	leveldbStatePrefix                      []byte = []byte{0x00, 0x12}
	leveldbOperationHashPrefix              []byte = []byte{0x00, 0x13}
	leveldbManifestHeightPrefix             []byte = []byte{0x00, 0x14}
)

type LeveldbStorage struct {
	*logging.Logging
	db   *leveldb.DB
	encs *encoder.Encoders
	enc  encoder.Encoder
}

func NewLeveldbStorage(db *leveldb.DB, encs *encoder.Encoders, enc encoder.Encoder) *LeveldbStorage {
	return &LeveldbStorage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "leveldb-storage")
		}),
		db:   db,
		encs: encs,
		enc:  enc,
	}
}

func NewMemStorage(encs *encoder.Encoders, enc encoder.Encoder) *LeveldbStorage {
	db, _ := leveldb.Open(leveldbStorage.NewMemStorage(), nil)
	return NewLeveldbStorage(db, encs, enc)
}

func (st *LeveldbStorage) SyncerStorage() (storage.SyncerStorage, error) {
	return NewLeveldbSyncerStorage(st), nil
}

func (st *LeveldbStorage) DB() *leveldb.DB {
	return st.db
}

func (st *LeveldbStorage) Encoder() encoder.Encoder {
	return st.enc
}

func (st *LeveldbStorage) Encoders() *encoder.Encoders {
	return st.encs
}

func (st *LeveldbStorage) LastBlock() (block.Block, error) {
	var raw []byte

	if err := st.iter(
		leveldbBlockHeightPrefix,
		func(_ []byte, value []byte) (bool, error) {
			raw = value
			return false, nil
		},
		false,
	); err != nil {
		return nil, err
	}

	if raw == nil {
		return nil, nil
	}

	h, err := st.loadHash(raw)
	if err != nil {
		return nil, err
	}

	return st.Block(h)
}

func (st *LeveldbStorage) get(key []byte) ([]byte, error) {
	b, err := st.db.Get(key, nil)

	return b, LeveldbWrapError(err)
}

func (st *LeveldbStorage) Block(h valuehash.Hash) (block.Block, error) {
	raw, err := st.get(leveldbBlockHashKey(h))
	if err != nil {
		return nil, err
	}

	return st.loadBlock(raw)
}

func (st *LeveldbStorage) BlockByHeight(height base.Height) (block.Block, error) {
	var bh valuehash.Hash

	if raw, err := st.get(leveldbBlockHeightKey(height)); err != nil {
		return nil, err
	} else if h, err := st.loadHash(raw); err != nil {
		return nil, err
	} else {
		bh = h
	}

	return st.Block(bh)
}

func (st *LeveldbStorage) Manifest(h valuehash.Hash) (block.Manifest, error) {
	raw, err := st.get(leveldbManifestKey(h))
	if err != nil {
		return nil, err
	}

	return st.loadManifest(raw)
}

func (st *LeveldbStorage) ManifestByHeight(height base.Height) (block.Manifest, error) {
	var bh valuehash.Hash

	if raw, err := st.get(leveldbBlockHeightKey(height)); err != nil {
		return nil, err
	} else if h, err := st.loadHash(raw); err != nil {
		return nil, err
	} else {
		bh = h
	}

	return st.Manifest(bh)
}

func (st *LeveldbStorage) loadLastVoteproof(stage base.Stage) (base.Voteproof, error) {
	return st.filterVoteproof(leveldbVoteproofHeightPrefix, stage)
}

func (st *LeveldbStorage) newVoteproof(voteproof base.Voteproof) error {
	st.Log().Debug().
		Hinted("height", voteproof.Height()).
		Hinted("round", voteproof.Round()).
		Hinted("stage", voteproof.Stage()).
		Msg("voteproof stored")

	raw, err := st.enc.Encode(voteproof)
	if err != nil {
		return err
	}

	hb := LeveldbDataWithEncoder(st.enc, raw)

	return LeveldbWrapError(st.db.Put(leveldbVoteproofKey(voteproof), hb, nil))
}

func (st *LeveldbStorage) LastINITVoteproof() (base.Voteproof, error) {
	return st.loadLastVoteproof(base.StageINIT)
}

func (st *LeveldbStorage) NewINITVoteproof(voteproof base.Voteproof) error {
	return st.newVoteproof(voteproof)
}

func (st *LeveldbStorage) filterVoteproof(prefix []byte, stage base.Stage) (base.Voteproof, error) {
	var raw []byte
	if err := st.iter(
		prefix,
		func(key, value []byte) (bool, error) {
			var height int64
			var round uint64
			var stg uint8
			n, err := fmt.Sscanf(
				string(key[len(leveldbVoteproofHeightPrefix):]),
				"%020d-%020d-%d", &height, &round, &stg,
			)
			if err != nil {
				return false, err
			}

			if n != 3 {
				return false, xerrors.Errorf("invalid formatted key found: key=%q", string(key))
			}

			if base.Stage(stg) != stage {
				return true, nil
			}

			raw = value
			return false, nil
		},
		false,
	); err != nil {
		return nil, err
	}

	return st.loadVoteproof(raw)
}

func (st *LeveldbStorage) LastINITVoteproofOfHeight(height base.Height) (base.Voteproof, error) {
	return st.filterVoteproof(leveldbVoteproofKeyByHeight(height), base.StageINIT)
}

func (st *LeveldbStorage) LastACCEPTVoteproofOfHeight(height base.Height) (base.Voteproof, error) {
	return st.filterVoteproof(leveldbVoteproofKeyByHeight(height), base.StageACCEPT)
}

func (st *LeveldbStorage) LastACCEPTVoteproof() (base.Voteproof, error) {
	return st.loadLastVoteproof(base.StageACCEPT)
}

func (st *LeveldbStorage) NewACCEPTVoteproof(voteproof base.Voteproof) error {
	return st.newVoteproof(voteproof)
}

func (st *LeveldbStorage) Voteproofs(callback func(base.Voteproof) (bool, error), sort bool) error {
	return st.iter(
		leveldbVoteproofHeightPrefix,
		func(_, value []byte) (bool, error) {
			voteproof, err := st.loadVoteproof(value)
			if err != nil {
				return false, err
			}

			return callback(voteproof)
		},
		sort,
	)
}

func (st *LeveldbStorage) sealKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(leveldbSealPrefix, h.Bytes())
}

func (st *LeveldbStorage) sealHashKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(leveldbSealHashPrefix, h.Bytes())
}

func (st *LeveldbStorage) newStagedOperationSealKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(
		leveldbStagedOperationSealPrefix,
		util.ULIDBytes(),
		[]byte("-"), // delimiter
		h.Bytes(),
	)
}

func (st *LeveldbStorage) newStagedOperationSealReverseKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(leveldbStagedOperationSealReversePrefix, h.Bytes())
}

func (st *LeveldbStorage) Seal(h valuehash.Hash) (seal.Seal, error) {
	return st.sealByKey(st.sealKey(h))
}

func (st *LeveldbStorage) sealByKey(key []byte) (seal.Seal, error) {
	b, err := st.get(key)
	if err != nil {
		return nil, err
	}

	return st.loadSeal(b)
}

func (st *LeveldbStorage) NewSeals(seals []seal.Seal) error {
	batch := &leveldb.Batch{}

	inserted := map[valuehash.Hash]struct{}{}
	for _, sl := range seals {
		if _, found := inserted[sl.Hash()]; found {
			continue
		}

		if err := st.newSeal(batch, sl); err != nil {
			return err
		}
		inserted[sl.Hash()] = struct{}{}
	}

	return LeveldbWrapError(st.db.Write(batch, nil))
}

func (st *LeveldbStorage) newSeal(batch *leveldb.Batch, sl seal.Seal) error {
	raw, err := st.enc.Encode(sl)
	if err != nil {
		return err
	}
	rawHash, err := st.enc.Encode(sl.Hash())
	if err != nil {
		return err
	}

	batch.Put(
		st.sealHashKey(sl.Hash()),
		LeveldbDataWithEncoder(st.enc, rawHash),
	)

	key := st.sealKey(sl.Hash())
	hb := LeveldbDataWithEncoder(st.enc, raw)
	if _, ok := sl.(operation.Seal); !ok {
		batch.Put(key, hb)
		return nil
	}

	batch.Put(key, hb)

	okey := st.newStagedOperationSealKey(sl.Hash())
	batch.Put(okey, key)
	batch.Put(st.newStagedOperationSealReverseKey(sl.Hash()), okey)

	return nil
}

func (st *LeveldbStorage) loadHinter(b []byte) (hint.Hinter, error) {
	if b == nil {
		return nil, nil
	}

	var ht hint.Hint
	ht, raw, err := LeveldbLoadHint(b)
	if err != nil {
		return nil, err
	}

	enc, err := st.encs.Encoder(ht.Type(), ht.Version())
	if err != nil {
		return nil, err
	}

	return enc.DecodeByHint(raw)
}

func (st *LeveldbStorage) loadVoteproof(b []byte) (base.Voteproof, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(base.Voteproof); !ok {
		return nil, xerrors.Errorf("not base.Voteproof: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *LeveldbStorage) loadBlock(b []byte) (block.Block, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(block.Block); !ok {
		return nil, xerrors.Errorf("not Block: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *LeveldbStorage) loadManifest(b []byte) (block.Manifest, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(block.Manifest); !ok {
		return nil, xerrors.Errorf("not Block: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *LeveldbStorage) loadSeal(b []byte) (seal.Seal, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(seal.Seal); !ok {
		return nil, xerrors.Errorf("not Seal: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *LeveldbStorage) loadHash(b []byte) (valuehash.Hash, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(valuehash.Hash); !ok {
		return nil, xerrors.Errorf("not Seal: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *LeveldbStorage) loadState(b []byte) (state.State, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(state.State); !ok {
		return nil, xerrors.Errorf("not state.State: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *LeveldbStorage) iter(
	prefix []byte,
	callback func([]byte /* key */, []byte /* value */) (bool, error),
	sort bool,
) error {
	iter := st.db.NewIterator(leveldbutil.BytesPrefix(prefix), nil)
	defer iter.Release()

	var seek func() bool
	var next func() bool
	if sort {
		seek = iter.First
		next = iter.Next
	} else {
		seek = iter.Last
		next = iter.Prev
	}

	if !seek() {
		return nil
	}

	for {
		if keep, err := callback(util.CopyBytes(iter.Key()), util.CopyBytes(iter.Value())); err != nil {
			return err
		} else if !keep {
			break
		}
		if !next() {
			break
		}
	}

	return LeveldbWrapError(iter.Error())
}

func (st *LeveldbStorage) Seals(callback func(valuehash.Hash, seal.Seal) (bool, error), sort bool, load bool) error {
	var prefix []byte
	var iterFunc func([]byte, []byte) (bool, error)

	if load {
		prefix = leveldbSealPrefix
		iterFunc = func(_, value []byte) (bool, error) {
			sl, err := st.loadSeal(value)
			if err != nil {
				return false, err
			}

			return callback(sl.Hash(), sl)
		}
	} else {
		prefix = leveldbSealHashPrefix
		iterFunc = func(_, value []byte) (bool, error) {
			h, err := st.loadHash(value)
			if err != nil {
				return false, err
			}

			return callback(h, nil)
		}
	}

	return st.iter(prefix, iterFunc, sort)
}

func (st *LeveldbStorage) StagedOperationSeals(callback func(operation.Seal) (bool, error), sort bool) error {
	return st.iter(
		leveldbStagedOperationSealPrefix,
		func(_, value []byte) (bool, error) {
			var osl operation.Seal
			if v, err := st.sealByKey(value); err != nil {
				return false, err
			} else if sl, ok := v.(operation.Seal); !ok {
				return false, xerrors.Errorf("not operation.Seal: %T", v)
			} else {
				osl = sl
			}
			return callback(osl)
		},
		sort,
	)
}

func (st *LeveldbStorage) UnstagedOperationSeals(seals []valuehash.Hash) error {
	batch := &leveldb.Batch{}

	if err := leveldbUnstageOperationSeals(st, batch, seals); err != nil {
		return err
	}

	return LeveldbWrapError(st.db.Write(batch, nil))
}

func (st *LeveldbStorage) Proposals(callback func(ballot.Proposal) (bool, error), sort bool) error {
	return st.iter(
		leveldbProposalPrefix,
		func(_, value []byte) (bool, error) {
			if sl, err := st.sealByKey(value); err != nil {
				return false, err
			} else if pr, ok := sl.(ballot.Proposal); !ok {
				return false, xerrors.Errorf("not Proposal: %T", sl)
			} else {
				return callback(pr)
			}
		},
		sort,
	)
}

func (st *LeveldbStorage) proposalKey(height base.Height, round base.Round) []byte {
	return util.ConcatBytesSlice(leveldbProposalPrefix, height.Bytes(), round.Bytes())
}

func (st *LeveldbStorage) NewProposal(proposal ballot.Proposal) error {
	sealKey := st.sealKey(proposal.Hash())
	if found, err := st.db.Has(sealKey, nil); err != nil {
		return LeveldbWrapError(err)
	} else if !found {
		if err := st.NewSeals([]seal.Seal{proposal}); err != nil {
			return err
		}
	}

	if err := st.db.Put(st.proposalKey(proposal.Height(), proposal.Round()), sealKey, nil); err != nil {
		return LeveldbWrapError(err)
	}

	return nil
}

func (st *LeveldbStorage) Proposal(height base.Height, round base.Round) (ballot.Proposal, error) {
	sealKey, err := st.get(st.proposalKey(height, round))
	if err != nil {
		return nil, err
	}

	sl, err := st.sealByKey(sealKey)
	if err != nil {
		return nil, err
	}

	return sl.(ballot.Proposal), nil
}

func (st *LeveldbStorage) State(key string) (state.State, bool, error) {
	b, err := st.get(leveldbStateKey(key))
	if err != nil {
		if xerrors.Is(err, storage.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	stt, err := st.loadState(b)

	return stt, st != nil, err
}

func (st *LeveldbStorage) NewState(sta state.State) error {
	if b, err := LeveldbMarshal(st.enc, sta); err != nil {
		return err
	} else if err := st.db.Put(leveldbStateKey(sta.Key()), b, nil); err != nil {
		return LeveldbWrapError(err)
	}

	return nil
}

func (st *LeveldbStorage) HasOperation(h valuehash.Hash) (bool, error) {
	found, err := st.db.Has(leveldbOperationHashKey(h), nil)

	return found, LeveldbWrapError(err)
}

func (st *LeveldbStorage) OpenBlockStorage(blk block.Block) (storage.BlockStorage, error) {
	return NewLeveldbBlockStorage(st, blk)
}

func leveldbBlockHeightKey(height base.Height) []byte {
	return util.ConcatBytesSlice(
		leveldbBlockHeightPrefix,
		[]byte(fmt.Sprintf("%020d", height.Int64())),
	)
}

func leveldbManifestHeightKey(height base.Height) []byte {
	return util.ConcatBytesSlice(
		leveldbManifestHeightPrefix,
		[]byte(fmt.Sprintf("%020d", height.Int64())),
	)
}

func leveldbBlockHashKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(
		leveldbBlockHashPrefix,
		h.Bytes(),
	)
}

func leveldbManifestKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(
		leveldbManifestPrefix,
		h.Bytes(),
	)
}

func leveldbVoteproofKey(voteproof base.Voteproof) []byte {
	return util.ConcatBytesSlice(
		leveldbVoteproofHeightPrefix,
		[]byte(fmt.Sprintf(
			"%020d-%020d-%d",
			voteproof.Height().Int64(),
			voteproof.Round().Uint64(),
			voteproof.Stage(),
		)),
	)
}

func leveldbVoteproofKeyByHeight(height base.Height) []byte {
	return util.ConcatBytesSlice(
		leveldbVoteproofHeightPrefix,
		[]byte(fmt.Sprintf("%020d-", height.Int64())),
	)
}

func leveldbBlockOperationsKey(blk block.Block) []byte {
	return util.ConcatBytesSlice(
		leveldbBlockOperationsPrefix,
		[]byte(fmt.Sprintf("%020d", blk.Height().Int64())),
	)
}

func leveldbBlockStatesKey(blk block.Block) []byte {
	return util.ConcatBytesSlice(
		leveldbBlockStatesPrefix,
		[]byte(fmt.Sprintf("%020d", blk.Height().Int64())),
	)
}

func leveldbStateKey(key string) []byte {
	return util.ConcatBytesSlice(
		leveldbStatePrefix,
		[]byte(key),
	)
}

func leveldbOperationHashKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(
		leveldbOperationHashPrefix,
		h.Bytes(),
	)
}

func leveldbUnstageOperationSeals(st *LeveldbStorage, batch *leveldb.Batch, seals []valuehash.Hash) error {
	if len(seals) < 1 {
		return nil
	}

	hashMap := map[string]struct{}{}
	for _, h := range seals {
		hashMap[h.String()] = struct{}{}
	}

	for _, h := range seals {
		rkey := st.newStagedOperationSealReverseKey(h)
		if key, err := st.get(rkey); err != nil {
			return err
		} else {
			batch.Delete(key)
			batch.Delete(rkey)
		}
	}

	return nil
}
