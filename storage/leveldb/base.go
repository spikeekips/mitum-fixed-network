package leveldbstorage

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
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
	keyPrefixTmp                            []byte = []byte{0x00, 0x00}
	keyPrefixBlockHeight                    []byte = []byte{0x00, 0x01}
	keyPrefixBlockHash                      []byte = []byte{0x00, 0x02}
	keyPrefixManifest                       []byte = []byte{0x00, 0x03}
	keyPrefixProposal                       []byte = []byte{0x00, 0x04}
	keyPrefixProposalFacts                  []byte = []byte{0x00, 0x05}
	keyPrefixBlockOperations                []byte = []byte{0x00, 0x06}
	keyPrefixBlockStates                    []byte = []byte{0x00, 0x07}
	keyPrefixState                          []byte = []byte{0x00, 0x08}
	keyPrefixOperationFactHash              []byte = []byte{0x00, 0x09}
	keyPrefixManifestHeight                 []byte = []byte{0x00, 0x10}
	keyPrefixINITVoteproof                  []byte = []byte{0x00, 0x11}
	keyPrefixACCEPTVoteproof                []byte = []byte{0x00, 0x12}
	keyPrefixBlockdataMap                   []byte = []byte{0x00, 0x13}
	keyPrefixInfo                           []byte = []byte{0x00, 0x14}
	keyPrefixStagedOperationFactHash        []byte = []byte{0x00, 0x15}
	keyPrefixStagedOperationFactHashReverse []byte = []byte{0x00, 0x16}
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
	raw, err := st.get(leveldbManifestKey(h))
	if err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	m, err := st.loadManifest(raw)
	if err != nil {
		return nil, false, err
	}

	return m, true, nil
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

func (st *Database) Manifests(load, reverse bool, limit int64, callback func(base.Height, valuehash.Hash, block.Manifest) (bool, error)) error {
	var counted int64
	return st.iter(
		keyPrefixManifestHeight,
		func(_, value []byte) (bool, error) {
			counted++

			m, err := st.loadManifest(value)
			if err != nil {
				return false, err
			}

			switch keep, err := callback(m.Height(), m.Hash(), m); {
			case err != nil:
				return false, err
			case !keep:
				return false, nil
			case counted == limit:
				return false, nil
			default:
				return true, nil
			}
		},
		!reverse,
	)
}

func (st *Database) newStagedOperationKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(
		keyPrefixStagedOperationFactHash,
		util.ULIDBytes(),
		[]byte("-"), // delimiter
		h.Bytes(),
	)
}

func (st *Database) newStagedOperationReverseKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(
		keyPrefixStagedOperationFactHashReverse,
		h.Bytes(),
	)
}

func (st *Database) proposalByKey(key []byte) (base.Proposal, bool, error) {
	b, err := st.get(key)
	if err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	if sl, err := st.loadProposal(b); err != nil {
		return nil, false, err
	} else {
		return sl, true, nil
	}
}

func (st *Database) NewOperationSeals(seals []operation.Seal) error {
	batch := &leveldb.Batch{}

	filter := st.newStagedOperationFilter()

	for i := range seals {
		if err := st.newOperations(batch, seals[i].Operations(), filter); err != nil {
			return err
		}
	}

	return mergeError(st.db.Write(batch, nil))
}

func (st *Database) newOperations(
	batch *leveldb.Batch,
	ops []operation.Operation,
	filter func(valuehash.Hash) (bool, error),
) error {
	for i := range ops {
		op := ops[i]

		switch ok, err := filter(op.Fact().Hash()); {
		case err != nil:
			return err
		case !ok:
			continue
		}

		if err := st.newOperation(batch, op); err != nil {
			return err
		}
	}

	return nil
}

func (st *Database) NewOperations(ops []operation.Operation) error {
	batch := &leveldb.Batch{}

	filter := st.newStagedOperationFilter()
	for i := range ops {
		op := ops[i]

		switch ok, err := filter(op.Fact().Hash()); {
		case err != nil:
			return err
		case !ok:
			continue
		}

		if err := st.newOperation(batch, op); err != nil {
			return err
		}
	}

	return mergeError(st.db.Write(batch, nil))
}

func (st *Database) newOperation(batch *leveldb.Batch, op operation.Operation) error {
	raw, err := st.enc.Marshal(op)
	if err != nil {
		return err
	}

	k := st.newStagedOperationKey(op.Fact().Hash())
	batch.Put(k, encodeWithEncoder(raw, st.enc))
	batch.Put(st.newStagedOperationReverseKey(op.Fact().Hash()), k)

	return nil
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

func (st *Database) loadProposal(b []byte) (base.Proposal, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(base.Proposal); !ok {
		return nil, errors.Errorf("not Proposal: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *Database) loadHash(b []byte) (valuehash.Hash, error) {
	var h valuehash.Bytes
	if err := st.loadValue(b, &h); err != nil {
		return nil, err
	} else if h.IsEmpty() {
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

func (st *Database) loadBlockdataMap(b []byte) (block.BlockdataMap, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(block.BlockdataMap); !ok {
		return nil, errors.Errorf("not block.BlockdataMap: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *Database) loadOperation(b []byte) (operation.Operation, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(operation.Operation); !ok {
		return nil, errors.Errorf("not operation.Operation: %T", hinter)
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

func (st *Database) HasStagedOperation(fact valuehash.Hash) (bool, error) {
	found, err := st.db.Has(st.newStagedOperationReverseKey(fact), nil)

	return found, mergeError(err)
}

func (st *Database) StagedOperationsByFact(facts []valuehash.Hash) ([]operation.Operation, error) {
	var ops []operation.Operation
	for i := range facts {
		b, err := st.get(st.newStagedOperationReverseKey(facts[i]))
		if err != nil {
			if errors.Is(err, util.NotFoundError) {
				continue
			}

			return nil, err
		}

		o, err := st.get(b)
		if errors.Is(err, util.NotFoundError) {
			continue
		}
		if err != nil {
			return nil, err
		}

		op, err := st.loadOperation(o)
		if err != nil {
			return nil, err
		}

		ops = append(ops, op)
	}

	return ops, nil
}

func (st *Database) StagedOperations(callback func(operation.Operation) (bool, error), sort bool) error {
	return st.iter(
		keyPrefixStagedOperationFactHash,
		func(_, value []byte) (bool, error) {
			op, err := st.loadOperation(value)
			if err != nil {
				return false, err
			}

			return callback(op)
		},
		sort,
	)
}

func (st *Database) UnstagedOperations(facts []valuehash.Hash) error {
	batch := &leveldb.Batch{}

	if err := leveldbUnstageOperations(st, batch, facts); err != nil {
		return err
	}

	return mergeError(st.db.Write(batch, nil))
}

func (st *Database) Proposals(callback func(base.Proposal) (bool, error), sort bool) error {
	return st.iter(
		keyPrefixProposal,
		func(_, value []byte) (bool, error) {
			if proposal, err := st.loadProposal(value); err != nil {
				return false, err
			} else {
				return callback(proposal)
			}
		},
		sort,
	)
}

func (st *Database) Proposal(h valuehash.Hash) (base.Proposal, bool, error) {
	return st.proposalByKey(st.proposalKey(h))
}

func (st *Database) proposalKey(h valuehash.Hash) []byte {
	return util.ConcatBytesSlice(keyPrefixProposal, h.Bytes())
}

func (st *Database) proposalFactsKey(height base.Height, round base.Round, proposer base.Address) []byte {
	return util.ConcatBytesSlice(keyPrefixProposalFacts, height.Bytes(), round.Bytes(), proposer.Bytes())
}

func (st *Database) NewProposal(proposal base.Proposal) error {
	fact := proposal.Fact()
	if fact.Stage() != base.StageProposal {
		return util.WrongTypeError.Errorf("not proposal SignedBallotFact: %T", proposal)
	}

	k := st.proposalKey(fact.Hash())
	if found, err := st.db.Has(k, nil); err != nil {
		return mergeError(err)
	} else if found {
		return nil
	}

	batch := &leveldb.Batch{}
	raw, err := st.enc.Marshal(proposal)
	if err != nil {
		return err
	}

	batch.Put(k, encodeWithEncoder(raw, st.enc))
	batch.Put(
		st.proposalFactsKey(fact.Height(), fact.Round(), fact.Proposer()),
		k,
	)

	return mergeError(st.db.Write(batch, nil))
}

func (st *Database) ProposalByPoint(height base.Height, round base.Round, proposer base.Address) (base.Proposal, bool, error) {
	k, err := st.get(st.proposalFactsKey(height, round, proposer))
	if err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return st.proposalByKey(k)
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

func (st *Database) BlockdataMap(height base.Height) (block.BlockdataMap, bool, error) {
	if raw, err := st.get(leveldbBlockdataMapKey(height)); err != nil {
		if errors.Is(err, util.NotFoundError) {
			return nil, false, nil
		}

		return nil, false, err
	} else if i, err := st.loadBlockdataMap(raw); err != nil {
		return nil, false, err
	} else {
		return i, true, nil
	}
}

func (st *Database) SetBlockdataMaps(bds []block.BlockdataMap) error {
	if len(bds) < 1 {
		return errors.Errorf("empty BlockdataMaps")
	}

	batch := new(leveldb.Batch)
	for i := range bds {
		bd := bds[i]
		if b, err := marshal(bd, st.enc); err != nil {
			return err
		} else {
			batch.Put(leveldbBlockdataMapKey(bd.Height()), b)
		}
	}

	return mergeError(st.db.Write(batch, nil))
}

func (st *Database) LocalBlockdataMapsByHeight(height base.Height, callback func(block.BlockdataMap) (bool, error)) error {
	return st.iter(
		keyPrefixBlockdataMap,
		func(_, value []byte) (bool, error) {
			switch bd, err := st.loadBlockdataMap(value); {
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

func (st *Database) newStagedOperationFilter() func(valuehash.Hash) (bool, error) {
	inserted := map[string]struct{}{}
	return func(h valuehash.Hash) (bool, error) {
		k := h.String()
		if _, found := inserted[k]; found {
			return false, nil
		}

		switch found, err := st.HasOperationFact(h); {
		case err != nil:
			return false, err
		case found:
			return false, nil
		}

		switch found, err := st.HasStagedOperation(h); {
		case err != nil:
			return false, err
		case found:
			return false, nil
		}

		inserted[k] = struct{}{}

		return true, nil
	}
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

func leveldbBlockdataMapKey(height base.Height) []byte {
	return util.ConcatBytesSlice(keyPrefixBlockdataMap, height.Bytes())
}

func leveldbUnstageOperations(st *Database, batch *leveldb.Batch, facts []valuehash.Hash) error {
	for i := range facts {
		k := st.newStagedOperationReverseKey(facts[i])
		switch found, err := st.db.Has(k, nil); {
		case err != nil:
			return err
		case !found:
			continue
		default:
			batch.Delete(k)
		}

		b, err := st.get(k)
		if err != nil {
			return err
		}
		batch.Delete(b)
	}

	return nil
}

func leveldbInfoKey(key string) []byte {
	return util.ConcatBytesSlice(
		keyPrefixInfo,
		[]byte(key),
	)
}
