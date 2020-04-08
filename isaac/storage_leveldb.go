package isaac

import (
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	leveldbStorage "github.com/syndtr/goleveldb/leveldb/storage"
	leveldbutil "github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/tree"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
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

func (st *LeveldbStorage) SyncerStorage() SyncerStorage {
	return NewLeveldbSyncerStorage(st)
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

func (st *LeveldbStorage) LastBlock() (Block, error) {
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

	return b, storage.LeveldbWrapError(err)
}

func (st *LeveldbStorage) Block(h valuehash.Hash) (Block, error) {
	raw, err := st.get(leveldbBlockHashKey(h))
	if err != nil {
		return nil, err
	}

	return st.loadBlock(raw)
}

func (st *LeveldbStorage) BlockByHeight(height Height) (Block, error) {
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

func (st *LeveldbStorage) Manifest(h valuehash.Hash) (Manifest, error) {
	raw, err := st.get(leveldbManifestKey(h))
	if err != nil {
		return nil, err
	}

	return st.loadManifest(raw)
}

func (st *LeveldbStorage) ManifestByHeight(height Height) (Manifest, error) {
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

func (st *LeveldbStorage) loadLastVoteproof(stage Stage) (Voteproof, error) {
	return st.filterVoteproof(leveldbVoteproofHeightPrefix, stage)
}

func (st *LeveldbStorage) newVoteproof(voteproof Voteproof) error {
	st.Log().Debug().
		Hinted("height", voteproof.Height()).
		Hinted("round", voteproof.Round()).
		Hinted("stage", voteproof.Stage()).
		Msg("voteproof stored")

	raw, err := st.enc.Encode(voteproof)
	if err != nil {
		return err
	}

	hb := storage.LeveldbDataWithEncoder(st.enc, raw)

	return storage.LeveldbWrapError(st.db.Put(leveldbVoteproofKey(voteproof), hb, nil))
}

func (st *LeveldbStorage) LastINITVoteproof() (Voteproof, error) {
	return st.loadLastVoteproof(StageINIT)
}

func (st *LeveldbStorage) NewINITVoteproof(voteproof Voteproof) error {
	return st.newVoteproof(voteproof)
}

func (st *LeveldbStorage) filterVoteproof(prefix []byte, stage Stage) (Voteproof, error) {
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

			if Stage(stg) != stage {
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

func (st *LeveldbStorage) LastINITVoteproofOfHeight(height Height) (Voteproof, error) {
	return st.filterVoteproof(leveldbVoteproofKeyByHeight(height), StageINIT)
}

func (st *LeveldbStorage) LastACCEPTVoteproofOfHeight(height Height) (Voteproof, error) {
	return st.filterVoteproof(leveldbVoteproofKeyByHeight(height), StageACCEPT)
}

func (st *LeveldbStorage) LastACCEPTVoteproof() (Voteproof, error) {
	return st.loadLastVoteproof(StageACCEPT)
}

func (st *LeveldbStorage) NewACCEPTVoteproof(voteproof Voteproof) error {
	return st.newVoteproof(voteproof)
}

func (st *LeveldbStorage) Voteproofs(callback func(Voteproof) (bool, error), sort bool) error {
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

	return storage.LeveldbWrapError(st.db.Write(batch, nil))
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
		storage.LeveldbDataWithEncoder(st.enc, rawHash),
	)

	key := st.sealKey(sl.Hash())
	hb := storage.LeveldbDataWithEncoder(st.enc, raw)
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
	ht, raw, err := storage.LeveldbLoadHint(b)
	if err != nil {
		return nil, err
	}

	enc, err := st.encs.Encoder(ht.Type(), ht.Version())
	if err != nil {
		return nil, err
	}

	return enc.DecodeByHint(raw)
}

func (st *LeveldbStorage) loadVoteproof(b []byte) (Voteproof, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(Voteproof); !ok {
		return nil, xerrors.Errorf("not Voteproof: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *LeveldbStorage) loadBlock(b []byte) (Block, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(Block); !ok {
		return nil, xerrors.Errorf("not Block: %T", hinter)
	} else {
		return i, nil
	}
}

func (st *LeveldbStorage) loadManifest(b []byte) (Manifest, error) {
	if hinter, err := st.loadHinter(b); err != nil {
		return nil, err
	} else if hinter == nil {
		return nil, nil
	} else if i, ok := hinter.(Manifest); !ok {
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

	return storage.LeveldbWrapError(iter.Error())
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

	return storage.LeveldbWrapError(st.db.Write(batch, nil))
}

func (st *LeveldbStorage) Proposals(callback func(Proposal) (bool, error), sort bool) error {
	return st.iter(
		leveldbProposalPrefix,
		func(_, value []byte) (bool, error) {
			if sl, err := st.sealByKey(value); err != nil {
				return false, err
			} else if pr, ok := sl.(Proposal); !ok {
				return false, xerrors.Errorf("not Proposal: %T", sl)
			} else {
				return callback(pr)
			}
		},
		sort,
	)
}

func (st *LeveldbStorage) proposalKey(height Height, round Round) []byte {
	return util.ConcatBytesSlice(leveldbProposalPrefix, height.Bytes(), round.Bytes())
}

func (st *LeveldbStorage) NewProposal(proposal Proposal) error {
	sealKey := st.sealKey(proposal.Hash())
	if found, err := st.db.Has(sealKey, nil); err != nil {
		return storage.LeveldbWrapError(err)
	} else if !found {
		if err := st.NewSeals([]seal.Seal{proposal}); err != nil {
			return err
		}
	}

	if err := st.db.Put(st.proposalKey(proposal.Height(), proposal.Round()), sealKey, nil); err != nil {
		return storage.LeveldbWrapError(err)
	}

	return nil
}

func (st *LeveldbStorage) Proposal(height Height, round Round) (Proposal, error) {
	sealKey, err := st.get(st.proposalKey(height, round))
	if err != nil {
		return nil, err
	}

	sl, err := st.sealByKey(sealKey)
	if err != nil {
		return nil, err
	}

	return sl.(Proposal), nil
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
	if b, err := storage.LeveldbMarshal(st.enc, sta); err != nil {
		return err
	} else if err := st.db.Put(leveldbStateKey(sta.Key()), b, nil); err != nil {
		return storage.LeveldbWrapError(err)
	}

	return nil
}

func (st *LeveldbStorage) HasOperation(h valuehash.Hash) (bool, error) {
	found, err := st.db.Has(leveldbOperationHashKey(h), nil)

	return found, storage.LeveldbWrapError(err)
}

func (st *LeveldbStorage) OpenBlockStorage(block Block) (BlockStorage, error) {
	return NewLeveldbBlockStorage(st, block)
}

type LeveldbBlockStorage struct {
	st         *LeveldbStorage
	block      Block
	operations *tree.AVLTree
	states     *tree.AVLTree
	batch      *leveldb.Batch
}

func NewLeveldbBlockStorage(st *LeveldbStorage, block Block) (*LeveldbBlockStorage, error) {
	bst := &LeveldbBlockStorage{
		st:    st,
		block: block,
		batch: &leveldb.Batch{},
	}

	return bst, nil
}

func (bst *LeveldbBlockStorage) Block() Block {
	return bst.block
}

func (bst *LeveldbBlockStorage) SetBlock(block Block) error {
	if bst.block.Height() != block.Height() {
		return xerrors.Errorf(
			"block has different height from initial block; initial=%d != block=%d",
			bst.block.Height(),
			block.Height(),
		)
	}

	if bst.block.Round() != block.Round() {
		return xerrors.Errorf(
			"block has different round from initial block; initial=%d != block=%d",
			bst.block.Round(),
			block.Round(),
		)
	}

	if b, err := storage.LeveldbMarshal(bst.st.enc, block); err != nil {
		return err
	} else {
		bst.batch.Put(leveldbBlockHashKey(block.Hash()), b)
	}

	if b, err := storage.LeveldbMarshal(bst.st.enc, block.Manifest()); err != nil {
		return err
	} else {
		key := leveldbManifestKey(block.Hash())
		bst.batch.Put(key, b)
	}

	if b, err := storage.LeveldbMarshal(bst.st.enc, block.Hash()); err != nil {
		return err
	} else {
		bst.batch.Put(leveldbBlockHeightKey(block.Height()), b)
	}

	if err := bst.setOperations(block.Operations()); err != nil {
		return err
	}

	if err := bst.setStates(block.States()); err != nil {
		return err
	}

	bst.block = block

	return nil
}

func (bst *LeveldbBlockStorage) setOperations(tr *tree.AVLTree) error {
	if tr == nil || tr.Empty() {
		return nil
	}

	if b, err := storage.LeveldbMarshal(bst.st.enc, tr); err != nil { // block 1st
		return err
	} else {
		bst.batch.Put(leveldbBlockOperationsKey(bst.block), b)
	}

	// store operation hashes
	if err := tr.Traverse(func(node tree.Node) (bool, error) {
		op := node.Immutable().(operation.OperationAVLNode).Operation()

		raw, err := bst.st.enc.Encode(op.Hash())
		if err != nil {
			return false, err
		}

		bst.batch.Put(
			leveldbOperationHashKey(op.Hash()),
			storage.LeveldbDataWithEncoder(bst.st.enc, raw),
		)

		return true, nil
	}); err != nil {
		return err
	}

	bst.operations = tr

	return nil
}

func (bst *LeveldbBlockStorage) setStates(tr *tree.AVLTree) error {
	if tr == nil || tr.Empty() {
		return nil
	}

	if b, err := storage.LeveldbMarshal(bst.st.enc, tr); err != nil { // block 1st
		return err
	} else {
		bst.batch.Put(leveldbBlockStatesKey(bst.block), b)
	}

	if err := tr.Traverse(func(node tree.Node) (bool, error) {
		var st state.State
		if s, ok := node.Immutable().(state.StateV0AVLNode); !ok {
			return false, xerrors.Errorf("not state.StateV0AVLNode: %T", node)
		} else {
			st = s.State()
		}

		if b, err := storage.LeveldbMarshal(bst.st.enc, st); err != nil {
			return false, err
		} else {
			bst.batch.Put(leveldbStateKey(st.Key()), b)
		}

		return true, nil
	}); err != nil {
		return err
	}

	bst.states = tr

	return nil
}

func (bst *LeveldbBlockStorage) UnstageOperationSeals(hs []valuehash.Hash) error {
	return leveldbUnstageOperationSeals(bst.st, bst.batch, hs)
}

func (bst *LeveldbBlockStorage) Commit() error {
	return storage.LeveldbWrapError(bst.st.db.Write(bst.batch, nil))
}

func leveldbBlockHeightKey(height Height) []byte {
	return util.ConcatBytesSlice(
		leveldbBlockHeightPrefix,
		[]byte(fmt.Sprintf("%020d", height.Int64())),
	)
}

func leveldbManifestHeightKey(height Height) []byte {
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

func leveldbVoteproofKey(voteproof Voteproof) []byte {
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

func leveldbVoteproofKeyByHeight(height Height) []byte {
	return util.ConcatBytesSlice(
		leveldbVoteproofHeightPrefix,
		[]byte(fmt.Sprintf("%020d-", height.Int64())),
	)
}

func leveldbBlockOperationsKey(block Block) []byte {
	return util.ConcatBytesSlice(
		leveldbBlockOperationsPrefix,
		[]byte(fmt.Sprintf("%020d", block.Height().Int64())),
	)
}

func leveldbBlockStatesKey(block Block) []byte {
	return util.ConcatBytesSlice(
		leveldbBlockStatesPrefix,
		[]byte(fmt.Sprintf("%020d", block.Height().Int64())),
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
