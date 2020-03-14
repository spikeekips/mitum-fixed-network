package isaac

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/syndtr/goleveldb/leveldb"
	leveldbErrors "github.com/syndtr/goleveldb/leveldb/errors"
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
	leveldbBlockHeightPrefix                []byte = []byte{0x00, 0x01}
	leveldbBlockHashPrefix                  []byte = []byte{0x00, 0x02}
	leveldbVoteproofHeightPrefix            []byte = []byte{0x00, 0x03}
	leveldbSealPrefix                       []byte = []byte{0x00, 0x04}
	leveldbProposalPrefix                   []byte = []byte{0x00, 0x05}
	leveldbBlockOperationsPrefix            []byte = []byte{0x00, 0x06}
	leveldbBlockStatesPrefix                []byte = []byte{0x00, 0x07}
	leveldbStagedOperationSealPrefix        []byte = []byte{0x00, 0x08}
	leveldbStagedOperationSealReversePrefix []byte = []byte{0x00, 0x09}
	leveldbStatePrefix                      []byte = []byte{0x00, 0x10}
)

type LeveldbStorage struct {
	*logging.Logger
	db   *leveldb.DB
	encs *encoder.Encoders
	enc  encoder.Encoder
}

func NewLeveldbStorage(db *leveldb.DB, encs *encoder.Encoders, enc encoder.Encoder) *LeveldbStorage {
	return &LeveldbStorage{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
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

func (st *LeveldbStorage) Encoder() encoder.Encoder {
	return st.enc
}

func (st *LeveldbStorage) Encoders() *encoder.Encoders {
	return st.encs
}

func (st *LeveldbStorage) LastBlock() (Block, error) {
	var key []byte

	if err := st.iter(
		leveldbBlockHeightPrefix,
		func(_ []byte, value []byte) (bool, error) {
			key = value
			return false, nil
		},
		false,
	); err != nil {
		return nil, err
	}

	if key == nil {
		return nil, nil
	}

	raw, err := st.get(key)
	if err != nil {
		return nil, err
	}

	return st.loadBlock(raw)
}

func (st *LeveldbStorage) get(key []byte) ([]byte, error) {
	b, err := st.db.Get(key, nil)

	return b, WrapLeveldbErorr(err)
}

func (st *LeveldbStorage) Block(h valuehash.Hash) (Block, error) {
	raw, err := st.get(leveldbBlockHashKey(h))
	if err != nil {
		return nil, err
	}

	return st.loadBlock(raw)
}

func (st *LeveldbStorage) BlockByHeight(height Height) (Block, error) {
	key, err := st.get(leveldbBlockHeightKey(height))
	if err != nil {
		return nil, err
	}

	raw, err := st.get(key)
	if err != nil {
		return nil, err
	}

	return st.loadBlock(raw)
}

func (st *LeveldbStorage) loadLastVoteproof(stage Stage) (Voteproof, error) {
	return st.filterVoteproof(leveldbVoteproofHeightPrefix, stage)
}

func (st *LeveldbStorage) newVoteproof(voteproof Voteproof) error {
	st.Log().Debug().
		Int64("height", voteproof.Height().Int64()).
		Uint64("round", voteproof.Round().Uint64()).
		Str("stage", voteproof.Stage().String()).
		Msg("voteproof stored")

	raw, err := st.enc.Encode(voteproof)
	if err != nil {
		return err
	}

	hb := storage.LeveldbDataWithEncoder(st.enc, raw)
	return st.db.Put(leveldbVoteproofKey(voteproof), hb, nil)
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
	return util.ConcatSlice([][]byte{leveldbSealPrefix, h.Bytes()})
}

func (st *LeveldbStorage) newStagedOperationSealKey(h valuehash.Hash) []byte {
	return util.ConcatSlice([][]byte{
		leveldbStagedOperationSealPrefix,
		util.ULIDBytes(),
		[]byte("-"), // delimiter
		h.Bytes(),
	})
}

func (st *LeveldbStorage) newStagedOperationSealReverseKey(h valuehash.Hash) []byte {
	return util.ConcatSlice([][]byte{
		leveldbStagedOperationSealReversePrefix,
		h.Bytes(),
	})
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

		if err := st.newSeals(batch, sl); err != nil {
			return err
		}
		inserted[sl.Hash()] = struct{}{}
	}

	return st.db.Write(batch, nil)
}

func (st *LeveldbStorage) newSeals(batch *leveldb.Batch, sl seal.Seal) error {
	raw, err := st.enc.Encode(sl)
	if err != nil {
		return err
	}

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

	return iter.Error()
}

func (st *LeveldbStorage) Seals(callback func(seal.Seal) (bool, error), sort bool) error {
	return st.iter(
		leveldbSealPrefix,
		func(_, value []byte) (bool, error) {
			sl, err := st.loadSeal(value)
			if err != nil {
				return false, err
			}

			return callback(sl)
		},
		sort,
	)
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
	return util.ConcatSlice([][]byte{leveldbProposalPrefix, height.Bytes(), round.Bytes()})
}

func (st *LeveldbStorage) NewProposal(proposal Proposal) error {
	sealKey := st.sealKey(proposal.Hash())
	if found, err := st.db.Has(sealKey, nil); err != nil {
		return err
	} else if !found {
		if err := st.NewSeals([]seal.Seal{proposal}); err != nil {
			return err
		}
	}

	if err := st.db.Put(st.proposalKey(proposal.Height(), proposal.Round()), sealKey, nil); err != nil {
		return err
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
	if b, err := st.marshal(sta); err != nil {
		return err
	} else if err := st.db.Put(leveldbStateKey(sta.Key()), b, nil); err != nil {
		return err
	}

	return nil
}

func (st *LeveldbStorage) OpenBlockStorage(block Block) (BlockStorage, error) {
	return NewLeveldbBlockStorage(st, block)
}

func (st *LeveldbStorage) marshal(i interface{}) ([]byte, error) {
	b, err := st.enc.Encode(i)
	if err != nil {
		return nil, err
	}

	return storage.LeveldbDataWithEncoder(st.enc, b), nil
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

	bst.block = block

	return nil
}

func (bst *LeveldbBlockStorage) SetOperations(tr *tree.AVLTree) error {
	if tr == nil {
		return nil
	}

	if b, err := bst.st.marshal(tr); err != nil { // block 1st
		return err
	} else {
		bst.batch.Put(leveldbBlockOperationsKey(bst.block), b)
	}

	bst.operations = tr

	return nil
}

func (bst *LeveldbBlockStorage) SetStates(tr *tree.AVLTree) error {
	if tr == nil {
		return nil
	}

	if b, err := bst.st.marshal(tr); err != nil { // block 1st
		return err
	} else {
		bst.batch.Put(leveldbBlockStatesKey(bst.block), b)
	}

	if err := tr.Traverse(func(node tree.Node) (bool, error) {
		var st state.State
		if s, ok := node.(*state.StateV0AVLNode); !ok {
			return false, xerrors.Errorf("not state.StateV0AVLNode: %T", node)
		} else {
			st = s.State()
		}

		if b, err := bst.st.marshal(st); err != nil {
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
	if len(hs) < 1 {
		return nil
	}

	hashMap := map[string]struct{}{}
	for _, h := range hs {
		hashMap[h.String()] = struct{}{}
	}

	for _, h := range hs {
		rkey := bst.st.newStagedOperationSealReverseKey(h)
		if key, err := bst.st.get(rkey); err != nil {
			return err
		} else {
			bst.batch.Delete(key)
			bst.batch.Delete(rkey)
		}
	}

	return nil
}

func (bst *LeveldbBlockStorage) Commit() error {
	if b, err := bst.st.marshal(bst.block); err != nil { // block 1st
		return err
	} else {
		key := leveldbBlockHashKey(bst.block.Hash())

		bst.batch.Put(leveldbBlockHeightKey(bst.block.Height()), key)
		bst.batch.Put(key, b)
	}

	return bst.st.db.Write(bst.batch, nil)
}

func leveldbBlockHeightKey(height Height) []byte {
	return util.ConcatSlice([][]byte{
		leveldbBlockHeightPrefix,
		[]byte(fmt.Sprintf("%020d", height.Int64())),
	})
}

func leveldbBlockHashKey(h valuehash.Hash) []byte {
	return util.ConcatSlice([][]byte{
		leveldbBlockHashPrefix,
		h.Bytes(),
	})
}

func leveldbVoteproofKey(voteproof Voteproof) []byte {
	return util.ConcatSlice([][]byte{
		leveldbVoteproofHeightPrefix,
		[]byte(fmt.Sprintf(
			"%020d-%020d-%d",
			voteproof.Height().Int64(),
			voteproof.Round().Uint64(),
			voteproof.Stage(),
		)),
	})
}

func leveldbVoteproofKeyByHeight(height Height) []byte {
	return util.ConcatSlice([][]byte{
		leveldbVoteproofHeightPrefix,
		[]byte(fmt.Sprintf("%020d-", height.Int64())),
	})
}

func leveldbBlockOperationsKey(block Block) []byte {
	return util.ConcatSlice([][]byte{
		leveldbBlockOperationsPrefix,
		[]byte(fmt.Sprintf("%020d", block.Height().Int64())),
	})
}

func leveldbBlockStatesKey(block Block) []byte {
	return util.ConcatSlice([][]byte{
		leveldbBlockStatesPrefix,
		[]byte(fmt.Sprintf("%020d", block.Height().Int64())),
	})
}

func leveldbStateKey(key string) []byte {
	return util.ConcatSlice([][]byte{leveldbStatePrefix, []byte(key)})
}

func WrapLeveldbErorr(err error) error {
	if err == nil {
		return nil
	}

	if err == leveldbErrors.ErrNotFound {
		return storage.NotFoundError.Wrap(err)
	}

	return err
}
