package leveldbstorage

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/syndtr/goleveldb/leveldb"
	leveldbStorage "github.com/syndtr/goleveldb/leveldb/storage"
	leveldbutil "github.com/syndtr/goleveldb/leveldb/util"
)

var (
	keyPrefixTmp                        []byte = []byte{0x00, 0x00}
	keyPrefixBlockHeight                []byte = []byte{0x00, 0x01}
	keyPrefixBlockHash                  []byte = []byte{0x00, 0x02}
	keyPrefixManifest                   []byte = []byte{0x00, 0x03}
	keyPrefixSeal                       []byte = []byte{0x00, 0x05}
	keyPrefixSealHash                   []byte = []byte{0x00, 0x06}
	keyPrefixProposal                   []byte = []byte{0x00, 0x07}
	keyPrefixBlockOperations            []byte = []byte{0x00, 0x08}
	keyPrefixBlockStates                []byte = []byte{0x00, 0x09}
	keyPrefixStagedOperationSeal        []byte = []byte{0x00, 0x10}
	keyPrefixStagedOperationSealReverse []byte = []byte{0x00, 0x11}
	keyPrefixState                      []byte = []byte{0x00, 0x12}
	keyPrefixOperationFactHash          []byte = []byte{0x00, 0x13}
	keyPrefixManifestHeight             []byte = []byte{0x00, 0x14}
	keyPrefixINITVoteproof              []byte = []byte{0x00, 0x15}
	keyPrefixACCEPTVoteproof            []byte = []byte{0x00, 0x16}
	keyPrefixBlockDataMap               []byte = []byte{0x00, 0x17}
	keyPrefixInfo                       []byte = []byte{0x00, 0x18}
)

type Database struct {
	*logging.Logging
	db   *leveldb.DB
	encs *encoder.Encoders
	enc  encoder.Encoder
}

func NewDatabase(db *leveldb.DB, encs *encoder.Encoders, enc encoder.Encoder) *Database {
	return &Database{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "leveldb-database")
		}),
		db:   db,
		encs: encs,
		enc:  enc,
	}
}

func NewMemDatabase(encs *encoder.Encoders, enc encoder.Encoder) *Database {
	db, _ := leveldb.Open(leveldbStorage.NewMemStorage(), nil)
	return NewDatabase(db, encs, enc)
}

func (st *Database) Initialize() error {
	return nil
}

func (st *Database) NewSyncerSession() (storage.SyncerSession, error) {
	return NewSyncerSession(st), nil
}

func (st *Database) DB() *leveldb.DB {
	return st.db
}

func (st *Database) Close() error {
	return st.db.Close()
}

func (st *Database) Clean() error {
	batch := &leveldb.Batch{}

	if err := st.iter(
		nil,
		func(key, _ []byte) (bool, error) {
			batch.Delete(key)

			return true, nil
		},
		false,
	); err != nil {
		return err
	}

	return mergeError(st.db.Write(batch, nil))
}

func (st *Database) CleanByHeight(height base.Height) error {
	if height <= base.PreGenesisHeight {
		return st.Clean()
	}

	// NOTE not perfectly working as intended :)
	batch := &leveldb.Batch{}

	h := height
end:
	for {
		switch m, found, err := st.ManifestByHeight(h); {
		case err != nil:
			return err
		case !found:
			break end
		default:
			batch.Delete(leveldbBlockHeightKey(h))
			batch.Delete(leveldbBlockHashKey(m.Hash()))
			batch.Delete(leveldbManifestHeightKey(h))
			batch.Delete(leveldbManifestKey(m.Hash()))
			batch.Delete(leveldbBlockOperationsKey(m))
			batch.Delete(leveldbBlockStatesKey(m))
		}

		h++
	}

	return mergeError(st.db.Write(batch, nil))
}

func (st *Database) Copy(source storage.Database) error {
	var sst *Database
	if s, ok := source.(*Database); !ok {
		return errors.Errorf("only leveldbstorage.Database can be allowed: %T", source)
	} else {
		sst = s
	}

	batch := &leveldb.Batch{}

	limit := 500
	if err := sst.iter(
		nil,
		func(key, value []byte) (bool, error) {
			batch.Put(key, value)

			if batch.Len() == limit {
				if err := mergeError(st.db.Write(batch, nil)); err != nil {
					return false, err
				}

				batch = &leveldb.Batch{}
			}

			return true, nil
		},
		false,
	); err != nil {
		return err
	}

	if batch.Len() < 1 {
		return nil
	}

	return mergeError(st.db.Write(batch, nil))
}

func (st *Database) Encoder() encoder.Encoder {
	return st.enc
}

func (st *Database) Encoders() *encoder.Encoders {
	return st.encs
}

func (st *Database) LastManifest() (block.Manifest, bool, error) {
	var raw []byte

	if err := st.iter(
		keyPrefixManifestHeight,
		func(_, value []byte) (bool, error) {
			raw = value
			return false, nil
		},
		false,
	); err != nil {
		return nil, false, err
	}

	if raw == nil {
		return nil, false, nil
	}

	h, err := st.loadHash(raw)
	if err != nil {
		return nil, false, err
	}

	return st.Manifest(h)
}

func (st *Database) lastBlock() (block.Block, bool, error) {
	var raw []byte

	if err := st.iter(
		keyPrefixBlockHeight,
		func(_, value []byte) (bool, error) {
			raw = value
			return false, nil
		},
		false,
	); err != nil {
		return nil, false, err
	}

	if raw == nil {
		return nil, false, nil
	}

	h, err := st.loadHash(raw)
	if err != nil {
		return nil, false, err
	}

	return st.block(h)
}

func (st *Database) get(key []byte) ([]byte, error) {
	b, err := st.db.Get(key, nil)

	return b, mergeError(err)
}

func (st *Database) block(h valuehash.Hash) (block.Block, bool, error) {
	if raw, err := st.get(leveldbBlockHashKey(h)); err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	} else if blk, err := st.loadBlock(raw); err != nil {
		return nil, false, err
	} else {
		return blk, true, nil
	}
}

func (st *Database) blockByHeight(height base.Height) (block.Block, bool, error) {
	if raw, err := st.get(leveldbBlockHeightKey(height)); err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	} else if h, err := st.loadHash(raw); err != nil {
		return nil, false, err
	} else {
		return st.block(h)
	}
}

func (st *Database) Manifest(h valuehash.Hash) (block.Manifest, bool, error) {
	if raw, err := st.get(leveldbManifestKey(h)); err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	} else if m, err := st.loadManifest(raw); err != nil {
		return nil, false, err
	} else {
		return m, true, nil
	}
}

func (st *Database) ManifestByHeight(height base.Height) (block.Manifest, bool, error) {
	if raw, err := st.get(leveldbBlockHeightKey(height)); err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	} else if h, err := st.loadHash(raw); err != nil {
		return nil, false, err
	} else {
		return st.Manifest(h)
	}
}

func (st *Database) sealKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(keyPrefixSeal, h.Bytes())
}

func (st *Database) sealHashKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(keyPrefixSealHash, h.Bytes())
}

func (st *Database) newStagedOperationSealKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(
		keyPrefixStagedOperationSeal,
		util.ULIDBytes(),
		[]byte("-"), // delimiter
		h.Bytes(),
	)
}

func (st *Database) newStagedOperationSealReverseKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(keyPrefixStagedOperationSealReverse, h.Bytes())
}

func (st *Database) Seal(h valuehash.Hash) (seal.Seal, bool, error) {
	return st.sealByKey(st.sealKey(h))
}

func (st *Database) sealByKey(key []byte) (seal.Seal, bool, error) {
	b, err := st.get(key)
	if err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	if sl, err := st.loadSeal(b); err != nil {
		return nil, false, err
	} else {
		return sl, true, nil
	}
}

func (st *Database) NewSeals(seals []seal.Seal) error {
	batch := &leveldb.Batch{}

	inserted := map[string]struct{}{}
	for _, sl := range seals {
		if _, found := inserted[sl.Hash().String()]; found {
			continue
		}

		if err := st.newSeal(batch, sl); err != nil {
			return err
		}
		inserted[sl.Hash().String()] = struct{}{}
	}

	return mergeError(st.db.Write(batch, nil))
}

func (st *Database) newSeal(batch *leveldb.Batch, sl seal.Seal) error {
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
		encodeWithEncoder(rawHash, st.enc),
	)

	key := st.sealKey(sl.Hash())
	hb := encodeWithEncoder(raw, st.enc)
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

func (st *Database) HasSeal(h valuehash.Hash) (bool, error) {
	found, err := st.db.Has(st.sealKey(h), nil)

	return found, mergeError(err)
}

func (st *Database) loadHinter(b []byte) (hint.Hinter, error) {
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

	return enc.Decode(raw)
}

func (st *Database) loadValue(b []byte, i interface{}) error {
	if b == nil {
		return nil
	}

	var ht hint.Hint
	ht, raw, err := loadHint(b)
	if err != nil {
		return err
	}

	enc, err := st.encs.Encoder(ht.Type(), ht.Version())
	if err != nil {
		return err
	}

	return enc.Unmarshal(raw, i)
}

func (st *Database) loadBlock(b []byte) (block.Block, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(block.Block); !ok {
		return nil, errors.Errorf("not Block: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *Database) loadManifest(b []byte) (block.Manifest, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(block.Manifest); !ok {
		return nil, errors.Errorf("not Block: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *Database) loadSeal(b []byte) (seal.Seal, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(seal.Seal); !ok {
		return nil, errors.Errorf("not Seal: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *Database) loadHash(b []byte) (valuehash.Hash, error) {
	var h valuehash.Bytes
	if err := st.loadValue(b, &h); err != nil {
		return nil, err
	} else if h.Empty() {
		return nil, errors.Errorf("empty hash found")
	}

	return h, nil
}

func (st *Database) loadState(b []byte) (state.State, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(state.State); !ok {
		return nil, errors.Errorf("not state.State: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *Database) loadBlockDataMap(b []byte) (block.BlockDataMap, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(block.BlockDataMap); !ok {
		return nil, errors.Errorf("not block.BlockDataMap: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *Database) iter(
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

	return mergeError(iter.Error())
}

func (st *Database) Seals(callback func(valuehash.Hash, seal.Seal) (bool, error), sort, load bool) error {
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

func (st *Database) SealsByHash(
	hashes []valuehash.Hash,
	callback func(valuehash.Hash, seal.Seal) (bool, error),
	_ bool,
) error {
	for _, h := range hashes {
		if sl, found, err := st.Seal(h); !found {
			continue
		} else if err != nil {
			return err
		} else if keep, err := callback(h, sl); err != nil {
			return err
		} else if !keep {
			break
		}
	}

	return nil
}

func (st *Database) StagedOperationSeals(callback func(operation.Seal) (bool, error), sort bool) error {
	return st.iter(
		keyPrefixStagedOperationSeal,
		func(_, value []byte) (bool, error) {
			var osl operation.Seal
			if v, found, err := st.sealByKey(value); err != nil || !found {
				return false, err
			} else if sl, ok := v.(operation.Seal); !ok {
				return false, errors.Errorf("not operation.Seal: %T", v)
			} else {
				osl = sl
			}
			return callback(osl)
		},
		sort,
	)
}

func (st *Database) UnstagedOperationSeals(seals []valuehash.Hash) error {
	batch := &leveldb.Batch{}

	if err := leveldbUnstageOperationSeals(st, batch, seals); err != nil {
		return err
	}

	return mergeError(st.db.Write(batch, nil))
}

func (st *Database) Proposals(callback func(ballot.Proposal) (bool, error), sort bool) error {
	return st.iter(
		keyPrefixProposal,
		func(_, value []byte) (bool, error) {
			if sl, found, err := st.sealByKey(value); err != nil || !found {
				return false, err
			} else if pr, ok := sl.(ballot.Proposal); !ok {
				return false, errors.Errorf("not Proposal: %T", sl)
			} else {
				return callback(pr)
			}
		},
		sort,
	)
}

func (st *Database) proposalKey(height base.Height, round base.Round, proposer base.Address) []byte {
	return util.ConcatBytesSlice(keyPrefixProposal, height.Bytes(), round.Bytes(), proposer.Bytes())
}

func (st *Database) NewProposal(proposal ballot.Proposal) error {
	sealKey := st.sealKey(proposal.Hash())
	if found, err := st.db.Has(sealKey, nil); err != nil {
		return mergeError(err)
	} else if !found {
		if err := st.NewSeals([]seal.Seal{proposal}); err != nil {
			return err
		}
	}

	if err := st.db.Put(st.proposalKey(proposal.Height(), proposal.Round(), proposal.Node()), sealKey, nil); err != nil {
		return mergeError(err)
	}

	return nil
}

func (st *Database) Proposal(height base.Height, round base.Round, proposer base.Address) (ballot.Proposal, bool, error) {
	sealKey, err := st.get(st.proposalKey(height, round, proposer))
	if err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	if sl, found, err := st.sealByKey(sealKey); err != nil || !found {
		return nil, false, err
	} else {
		return sl.(ballot.Proposal), true, nil
	}
}

func (st *Database) State(key string) (state.State, bool, error) {
	b, err := st.get(leveldbStateKey(key))
	if err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	stt, err := st.loadState(b)

	return stt, st != nil, err
}

func (st *Database) NewState(sta state.State) error {
	if b, err := marshal(sta, st.enc); err != nil {
		return err
	} else if err := st.db.Put(leveldbStateKey(sta.Key()), b, nil); err != nil {
		return mergeError(err)
	}

	return nil
}

func (st *Database) HasOperationFact(h valuehash.Hash) (bool, error) {
	found, err := st.db.Has(leveldbOperationFactHashKey(h), nil)

	return found, mergeError(err)
}

func (st *Database) NewSession(blk block.Block) (storage.DatabaseSession, error) {
	return NewSession(st, blk)
}

func (st *Database) SetInfo(key string, b []byte) error {
	if err := st.db.Put(leveldbInfoKey(key), b, nil); err != nil {
		return mergeError(err)
	}

	return nil
}

func (st *Database) Info(key string) ([]byte, bool, error) {
	if b, err := st.get(leveldbInfoKey(key)); err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	} else {
		return b, true, nil
	}
}

func (st *Database) LastVoteproof(stage base.Stage) base.Voteproof {
	var prefix []byte
	switch stage {
	case base.StageINIT:
		prefix = keyPrefixINITVoteproof
	case base.StageACCEPT:
		prefix = keyPrefixACCEPTVoteproof
	default:
		return nil
	}

	var raw []byte
	if err := st.iter(
		prefix,
		func(_, value []byte) (bool, error) {
			raw = value
			return false, nil
		},
		false,
	); err != nil {
		return nil
	}

	if raw == nil {
		return nil
	}

	if i, err := st.loadHinter(raw); err != nil {
		return nil
	} else if j, ok := i.(base.Voteproof); !ok {
		return nil
	} else {
		return j
	}
}

func (st *Database) Voteproof(height base.Height, stage base.Stage) (base.Voteproof, error) {
	var raw []byte
	if b, err := st.get(leveldbVoteproofKey(height, stage)); err != nil {
		return nil, err
	} else {
		raw = b
	}

	if raw == nil {
		return nil, nil
	}

	if i, err := st.loadHinter(raw); err != nil {
		return nil, err
	} else if j, ok := i.(base.Voteproof); !ok {
		return nil, errors.Errorf("wrong voteproof, not %t", i)
	} else {
		return j, nil
	}
}

func (st *Database) BlockDataMap(height base.Height) (block.BlockDataMap, bool, error) {
	if raw, err := st.get(leveldbBlockDataMapKey(height)); err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	} else if i, err := st.loadBlockDataMap(raw); err != nil {
		return nil, false, err
	} else {
		return i, true, nil
	}
}

func (st *Database) SetBlockDataMaps(bds []block.BlockDataMap) error {
	if len(bds) < 1 {
		return errors.Errorf("empty BlockDataMaps")
	}

	batch := new(leveldb.Batch)
	for i := range bds {
		bd := bds[i]
		if b, err := marshal(bd, st.enc); err != nil {
			return err
		} else {
			batch.Put(leveldbBlockDataMapKey(bd.Height()), b)
		}
	}

	return mergeError(st.db.Write(batch, nil))
}

func (st *Database) LocalBlockDataMapsByHeight(height base.Height, callback func(block.BlockDataMap) (bool, error)) error {
	return st.iter(
		keyPrefixBlockDataMap,
		func(_, value []byte) (bool, error) {
			switch bd, err := st.loadBlockDataMap(value); {
			case err != nil:
				return false, err
			case bd.Height() < height:
				return true, nil
			case !bd.IsLocal():
				return true, nil
			default:
				return callback(bd)
			}
		},
		true,
	)
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

func leveldbBlockOperationsKey(blk block.Manifest) []byte {
	return util.ConcatBytesSlice(
		keyPrefixBlockOperations,
		[]byte(fmt.Sprintf("%020d", blk.Height().Int64())),
	)
}

func leveldbBlockStatesKey(blk block.Manifest) []byte {
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

func leveldbOperationFactHashKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(
		keyPrefixOperationFactHash,
		h.Bytes(),
	)
}

func leveldbVoteproofKey(height base.Height, stage base.Stage) []byte {
	var prefix []byte
	switch stage {
	case base.StageINIT:
		prefix = keyPrefixINITVoteproof
	case base.StageACCEPT:
		prefix = keyPrefixACCEPTVoteproof
	default:
		return nil
	}

	return util.ConcatBytesSlice(
		prefix,
		[]byte(fmt.Sprintf("%020d", height.Int64())),
	)
}

func leveldbBlockDataMapKey(height base.Height) []byte {
	return util.ConcatBytesSlice(keyPrefixBlockDataMap, height.Bytes())
}

func leveldbUnstageOperationSeals(st *Database, batch *leveldb.Batch, seals []valuehash.Hash) error {
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

func leveldbInfoKey(key string) []byte {
	return util.ConcatBytesSlice(
		keyPrefixInfo,
		[]byte(key),
	)
}
