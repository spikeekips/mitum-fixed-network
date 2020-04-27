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
	keyPrefixTmp                        []byte = []byte{0x00, 0x00}
	keyPrefixBlockHeight                []byte = []byte{0x00, 0x01}
	keyPrefixBlockHash                  []byte = []byte{0x00, 0x02}
	keyPrefixManifest                   []byte = []byte{0x00, 0x03}
	keyPrefixVoteproofHeight            []byte = []byte{0x00, 0x04}
	keyPrefixSeal                       []byte = []byte{0x00, 0x05}
	keyPrefixSealHash                   []byte = []byte{0x00, 0x06}
	keyPrefixProposal                   []byte = []byte{0x00, 0x07}
	keyPrefixBlockOperations            []byte = []byte{0x00, 0x08}
	keyPrefixBlockStates                []byte = []byte{0x00, 0x09}
	keyPrefixStagedOperationSeal        []byte = []byte{0x00, 0x10}
	keyPrefixStagedOperationSealReverse []byte = []byte{0x00, 0x11}
	keyPrefixState                      []byte = []byte{0x00, 0x12}
	keyPrefixOperationHash              []byte = []byte{0x00, 0x13}
	keyPrefixManifestHeight             []byte = []byte{0x00, 0x14}
)

type Storage struct {
	*logging.Logging
	db   *leveldb.DB
	encs *encoder.Encoders
	enc  encoder.Encoder
}

func NewStorage(db *leveldb.DB, encs *encoder.Encoders, enc encoder.Encoder) *Storage {
	return &Storage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "leveldb-storage")
		}),
		db:   db,
		encs: encs,
		enc:  enc,
	}
}

func NewMemStorage(encs *encoder.Encoders, enc encoder.Encoder) *Storage {
	db, _ := leveldb.Open(leveldbStorage.NewMemStorage(), nil)
	return NewStorage(db, encs, enc)
}

func (st *Storage) SyncerStorage() (storage.SyncerStorage, error) {
	return NewSyncerStorage(st), nil
}

func (st *Storage) DB() *leveldb.DB {
	return st.db
}

func (st *Storage) Close() error {
	return st.db.Close()
}

func (st *Storage) Encoder() encoder.Encoder {
	return st.enc
}

func (st *Storage) Encoders() *encoder.Encoders {
	return st.encs
}

func (st *Storage) LastBlock() (block.Block, error) {
	var raw []byte

	if err := st.iter(
		keyPrefixBlockHeight,
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

func (st *Storage) get(key []byte) ([]byte, error) {
	b, err := st.db.Get(key, nil)

	return b, wrapError(err)
}

func (st *Storage) Block(h valuehash.Hash) (block.Block, error) {
	raw, err := st.get(leveldbBlockHashKey(h))
	if err != nil {
		return nil, err
	}

	return st.loadBlock(raw)
}

func (st *Storage) BlockByHeight(height base.Height) (block.Block, error) {
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

func (st *Storage) Manifest(h valuehash.Hash) (block.Manifest, error) {
	raw, err := st.get(leveldbManifestKey(h))
	if err != nil {
		return nil, err
	}

	return st.loadManifest(raw)
}

func (st *Storage) ManifestByHeight(height base.Height) (block.Manifest, error) {
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

func (st *Storage) loadLastVoteproof(stage base.Stage) (base.Voteproof, error) {
	return st.filterVoteproof(keyPrefixVoteproofHeight, stage)
}

func (st *Storage) newVoteproof(voteproof base.Voteproof) error {
	st.Log().Debug().
		Hinted("height", voteproof.Height()).
		Hinted("round", voteproof.Round()).
		Hinted("stage", voteproof.Stage()).
		Msg("voteproof stored")

	raw, err := st.enc.Marshal(voteproof)
	if err != nil {
		return err
	}

	hb := encodeWithEncoder(st.enc, raw)

	return wrapError(st.db.Put(leveldbVoteproofKey(voteproof), hb, nil))
}

func (st *Storage) LastINITVoteproof() (base.Voteproof, error) {
	return st.loadLastVoteproof(base.StageINIT)
}

func (st *Storage) NewINITVoteproof(voteproof base.Voteproof) error {
	return st.newVoteproof(voteproof)
}

func (st *Storage) filterVoteproof(prefix []byte, stage base.Stage) (base.Voteproof, error) {
	var raw []byte
	if err := st.iter(
		prefix,
		func(key, value []byte) (bool, error) {
			var height int64
			var round uint64
			var stg uint8
			n, err := fmt.Sscanf(
				string(key[len(keyPrefixVoteproofHeight):]),
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

func (st *Storage) LastINITVoteproofOfHeight(height base.Height) (base.Voteproof, error) {
	return st.filterVoteproof(leveldbVoteproofKeyByHeight(height), base.StageINIT)
}

func (st *Storage) LastACCEPTVoteproofOfHeight(height base.Height) (base.Voteproof, error) {
	return st.filterVoteproof(leveldbVoteproofKeyByHeight(height), base.StageACCEPT)
}

func (st *Storage) LastACCEPTVoteproof() (base.Voteproof, error) {
	return st.loadLastVoteproof(base.StageACCEPT)
}

func (st *Storage) NewACCEPTVoteproof(voteproof base.Voteproof) error {
	return st.newVoteproof(voteproof)
}

func (st *Storage) Voteproofs(callback func(base.Voteproof) (bool, error), sort bool) error {
	return st.iter(
		keyPrefixVoteproofHeight,
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

func (st *Storage) sealKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(keyPrefixSeal, h.Bytes())
}

func (st *Storage) sealHashKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(keyPrefixSealHash, h.Bytes())
}

func (st *Storage) newStagedOperationSealKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(
		keyPrefixStagedOperationSeal,
		util.ULIDBytes(),
		[]byte("-"), // delimiter
		h.Bytes(),
	)
}

func (st *Storage) newStagedOperationSealReverseKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(keyPrefixStagedOperationSealReverse, h.Bytes())
}

func (st *Storage) Seal(h valuehash.Hash) (seal.Seal, error) {
	return st.sealByKey(st.sealKey(h))
}

func (st *Storage) sealByKey(key []byte) (seal.Seal, error) {
	b, err := st.get(key)
	if err != nil {
		return nil, err
	}

	return st.loadSeal(b)
}

func (st *Storage) NewSeals(seals []seal.Seal) error {
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

	return wrapError(st.db.Write(batch, nil))
}

func (st *Storage) newSeal(batch *leveldb.Batch, sl seal.Seal) error {
	raw, err := st.enc.Marshal(sl)
	if err != nil {
		return err
	}
	rawHash, err := st.enc.Marshal(sl.Hash())
	if err != nil {
		return err
	}

	batch.Put(
		st.sealHashKey(sl.Hash()),
		encodeWithEncoder(st.enc, rawHash),
	)

	key := st.sealKey(sl.Hash())
	hb := encodeWithEncoder(st.enc, raw)
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

func (st *Storage) loadHinter(b []byte) (hint.Hinter, error) {
	if b == nil {
		return nil, nil
	}

	var ht hint.Hint
	ht, raw, err := loadHint(b)
	if err != nil {
		return nil, err
	}

	enc, err := st.encs.Encoder(ht.Type(), ht.Version())
	if err != nil {
		return nil, err
	}

	return enc.DecodeByHint(raw)
}

func (st *Storage) loadVoteproof(b []byte) (base.Voteproof, error) {
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

func (st *Storage) loadBlock(b []byte) (block.Block, error) {
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

func (st *Storage) loadManifest(b []byte) (block.Manifest, error) {
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

func (st *Storage) loadSeal(b []byte) (seal.Seal, error) {
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

func (st *Storage) loadHash(b []byte) (valuehash.Hash, error) {
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

func (st *Storage) loadState(b []byte) (state.State, error) {
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

func (st *Storage) iter(
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

	return wrapError(iter.Error())
}

func (st *Storage) Seals(callback func(valuehash.Hash, seal.Seal) (bool, error), sort bool, load bool) error {
	var prefix []byte
	var iterFunc func([]byte, []byte) (bool, error)

	if load {
		prefix = keyPrefixSeal
		iterFunc = func(_, value []byte) (bool, error) {
			sl, err := st.loadSeal(value)
			if err != nil {
				return false, err
			}

			return callback(sl.Hash(), sl)
		}
	} else {
		prefix = keyPrefixSealHash
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

func (st *Storage) StagedOperationSeals(callback func(operation.Seal) (bool, error), sort bool) error {
	return st.iter(
		keyPrefixStagedOperationSeal,
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

func (st *Storage) UnstagedOperationSeals(seals []valuehash.Hash) error {
	batch := &leveldb.Batch{}

	if err := leveldbUnstageOperationSeals(st, batch, seals); err != nil {
		return err
	}

	return wrapError(st.db.Write(batch, nil))
}

func (st *Storage) Proposals(callback func(ballot.Proposal) (bool, error), sort bool) error {
	return st.iter(
		keyPrefixProposal,
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

func (st *Storage) proposalKey(height base.Height, round base.Round) []byte {
	return util.ConcatBytesSlice(keyPrefixProposal, height.Bytes(), round.Bytes())
}

func (st *Storage) NewProposal(proposal ballot.Proposal) error {
	sealKey := st.sealKey(proposal.Hash())
	if found, err := st.db.Has(sealKey, nil); err != nil {
		return wrapError(err)
	} else if !found {
		if err := st.NewSeals([]seal.Seal{proposal}); err != nil {
			return err
		}
	}

	if err := st.db.Put(st.proposalKey(proposal.Height(), proposal.Round()), sealKey, nil); err != nil {
		return wrapError(err)
	}

	return nil
}

func (st *Storage) Proposal(height base.Height, round base.Round) (ballot.Proposal, error) {
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

func (st *Storage) State(key string) (state.State, bool, error) {
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

func (st *Storage) NewState(sta state.State) error {
	if b, err := marshal(st.enc, sta); err != nil {
		return err
	} else if err := st.db.Put(leveldbStateKey(sta.Key()), b, nil); err != nil {
		return wrapError(err)
	}

	return nil
}

func (st *Storage) HasOperation(h valuehash.Hash) (bool, error) {
	found, err := st.db.Has(leveldbOperationHashKey(h), nil)

	return found, wrapError(err)
}

func (st *Storage) OpenBlockStorage(blk block.Block) (storage.BlockStorage, error) {
	return NewBlockStorage(st, blk)
}

func leveldbBlockHeightKey(height base.Height) []byte {
	return util.ConcatBytesSlice(
		keyPrefixBlockHeight,
		[]byte(fmt.Sprintf("%020d", height.Int64())),
	)
}

func leveldbManifestHeightKey(height base.Height) []byte {
	return util.ConcatBytesSlice(
		keyPrefixManifestHeight,
		[]byte(fmt.Sprintf("%020d", height.Int64())),
	)
}

func leveldbBlockHashKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(
		keyPrefixBlockHash,
		h.Bytes(),
	)
}

func leveldbManifestKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(
		keyPrefixManifest,
		h.Bytes(),
	)
}

func leveldbVoteproofKey(voteproof base.Voteproof) []byte {
	return util.ConcatBytesSlice(
		keyPrefixVoteproofHeight,
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
		keyPrefixVoteproofHeight,
		[]byte(fmt.Sprintf("%020d-", height.Int64())),
	)
}

func leveldbBlockOperationsKey(blk block.Block) []byte {
	return util.ConcatBytesSlice(
		keyPrefixBlockOperations,
		[]byte(fmt.Sprintf("%020d", blk.Height().Int64())),
	)
}

func leveldbBlockStatesKey(blk block.Block) []byte {
	return util.ConcatBytesSlice(
		keyPrefixBlockStates,
		[]byte(fmt.Sprintf("%020d", blk.Height().Int64())),
	)
}

func leveldbStateKey(key string) []byte {
	return util.ConcatBytesSlice(
		keyPrefixState,
		[]byte(key),
	)
}

func leveldbOperationHashKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(
		keyPrefixOperationHash,
		h.Bytes(),
	)
}

func leveldbUnstageOperationSeals(st *Storage, batch *leveldb.Batch, seals []valuehash.Hash) error {
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
