package isaac

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/spikeekips/avl"
	"github.com/syndtr/goleveldb/leveldb"
	leveldbStorage "github.com/syndtr/goleveldb/leveldb/storage"
	leveldbutil "github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

var (
	leveldbBlockHeightPrefix     []byte = []byte{0x00, 0x01}
	leveldbBlockHashPrefix       []byte = []byte{0x00, 0x02}
	leveldbVoteproofHeightPrefix []byte = []byte{0x00, 0x03}
	leveldbSealPrefix            []byte = []byte{0x00, 0x04}
	leveldbProposalPrefix        []byte = []byte{0x00, 0x05}
	leveldbBlockOperationsPrefix []byte = []byte{0x00, 0x06}
	leveldbBlockStatesPrefix     []byte = []byte{0x00, 0x07}
)

type LeveldbStorage struct {
	*logging.Logger
	db         *leveldb.DB
	encs       *encoder.Encoders
	defaultEnc encoder.Encoder
}

func NewLeveldbStorage(db *leveldb.DB, encs *encoder.Encoders, defaultEnc encoder.Encoder) *LeveldbStorage {
	return &LeveldbStorage{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "leveldb-storage")
		}),
		db:         db,
		encs:       encs,
		defaultEnc: defaultEnc,
	}
}

func NewMemStorage(encs *encoder.Encoders, enc encoder.Encoder) *LeveldbStorage {
	db, _ := leveldb.Open(leveldbStorage.NewMemStorage(), nil)
	return NewLeveldbStorage(db, encs, enc)
}

func (st *LeveldbStorage) Encoder() encoder.Encoder {
	return st.defaultEnc
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

	raw, err := st.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	return st.loadBlock(raw)
}

func (st *LeveldbStorage) Block(h valuehash.Hash) (Block, error) {
	raw, err := st.db.Get(leveldbBlockHashKey(h), nil)
	if err != nil {
		return nil, err
	}

	return st.loadBlock(raw)
}

func (st *LeveldbStorage) BlockByHeight(height Height) (Block, error) {
	key, err := st.db.Get(leveldbBlockHeightKey(height), nil)
	if err != nil {
		return nil, err
	}

	raw, err := st.db.Get(key, nil)
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

	raw, err := st.defaultEnc.Encode(voteproof)
	if err != nil {
		return err
	}

	hb := storage.LeveldbDataWithEncoder(st.defaultEnc, raw)
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

func (st *LeveldbStorage) Seal(h valuehash.Hash) (seal.Seal, error) {
	return st.sealByKey(st.sealKey(h))
}

func (st *LeveldbStorage) sealByKey(key []byte) (seal.Seal, error) {
	raw, err := st.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	var ht hint.Hint
	ht, data, err := storage.LeveldbLoadHint(raw)
	if err != nil {
		return nil, err
	}

	enc, err := st.encs.Encoder(ht.Type(), ht.Version())
	if err != nil {
		return nil, err
	}

	hinter, err := enc.DecodeByHint(data)
	if err != nil {
		return nil, err
	}

	return hinter.(seal.Seal), nil
}

func (st *LeveldbStorage) NewSeal(sl seal.Seal) error {
	raw, err := st.defaultEnc.Encode(sl)
	if err != nil {
		return err
	}

	hb := storage.LeveldbDataWithEncoder(st.defaultEnc, raw)

	return st.db.Put(st.sealKey(sl.Hash()), hb, nil)
}

func (st *LeveldbStorage) loadVoteproof(b []byte) (Voteproof, error) {
	if b == nil {
		return nil, nil
	}

	var ht hint.Hint
	ht, data, err := storage.LeveldbLoadHint(b)
	if err != nil {
		return nil, err
	}

	enc, err := st.encs.Encoder(ht.Type(), ht.Version())
	if err != nil {
		return nil, err
	}

	var voteproof Voteproof
	if hinter, err := enc.DecodeByHint(data); err != nil {
		return nil, err
	} else if i, ok := hinter.(Voteproof); !ok {
		return nil, xerrors.Errorf("not Voteproof: %T", hinter)
	} else {
		voteproof = i
	}

	return voteproof, nil
}

func (st *LeveldbStorage) loadBlock(b []byte) (Block, error) {
	if b == nil {
		return nil, nil
	}

	var ht hint.Hint
	ht, data, err := storage.LeveldbLoadHint(b)
	if err != nil {
		return nil, err
	}

	enc, err := st.encs.Encoder(ht.Type(), ht.Version())
	if err != nil {
		return nil, err
	}

	var block Block
	if hinter, err := enc.DecodeByHint(data); err != nil {
		return nil, err
	} else if bl, ok := hinter.(Block); !ok {
		return nil, xerrors.Errorf("not Block: %T", hinter)
	} else {
		block = bl
	}

	return block, nil
}

func (st *LeveldbStorage) loadSeal(b []byte) (seal.Seal, error) {
	if b == nil {
		return nil, nil
	}

	var ht hint.Hint
	ht, data, err := storage.LeveldbLoadHint(b)
	if err != nil {
		return nil, err
	}

	enc, err := st.encs.Encoder(ht.Type(), ht.Version())
	if err != nil {
		return nil, err
	}

	hinter, err := enc.DecodeByHint(data)
	if err != nil {
		return nil, err
	}

	sl, ok := hinter.(seal.Seal)
	if !ok {
		return nil, xerrors.Errorf("not Seal: %T", hinter)
	}

	return sl, nil
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
		if err := st.NewSeal(proposal); err != nil {
			return err
		}
	}

	if err := st.db.Put(st.proposalKey(proposal.Height(), proposal.Round()), sealKey, nil); err != nil {
		return err
	}

	return nil
}

func (st *LeveldbStorage) Proposal(height Height, round Round) (Proposal, error) {
	sealKey, err := st.db.Get(st.proposalKey(height, round), nil)
	if err != nil {
		return nil, err
	}

	sl, err := st.sealByKey(sealKey)
	if err != nil {
		return nil, err
	}

	return sl.(Proposal), nil
}

func (st *LeveldbStorage) OpenBlockStorage(block Block) (BlockStorage, error) {
	return NewLeveldbBlockStorage(st, block)
}

type LeveldbBlockStorage struct {
	st         *LeveldbStorage
	block      Block
	operations *avl.Tree
	states     *avl.Tree
	batch      *leveldb.Batch
}

func NewLeveldbBlockStorage(st *LeveldbStorage, block Block) (*LeveldbBlockStorage, error) {
	bst := &LeveldbBlockStorage{
		st:    st,
		block: block,
		batch: &leveldb.Batch{},
	}

	if b, err := bst.marshal(block); err != nil { // block 1st
		return nil, err
	} else {
		key := leveldbBlockHashKey(block.Hash())
		bst.batch.Put(leveldbBlockHeightKey(block.Height()), key)
		bst.batch.Put(key, b)
	}

	return bst, nil
}

func (bst *LeveldbBlockStorage) Block() Block {
	return bst.block
}

func (bst *LeveldbBlockStorage) SetOperations(tree *avl.Tree) error {
	if b, err := bst.marshal(tree); err != nil { // block 1st
		return err
	} else {
		bst.batch.Put(leveldbBlockOperationsKey(bst.block), b)
	}

	bst.operations = tree

	return nil
}

func (bst *LeveldbBlockStorage) SetStates(tree *avl.Tree) error {
	if b, err := bst.marshal(tree); err != nil { // block 1st
		return err
	} else {
		bst.batch.Put(leveldbBlockStatesKey(bst.block), b)
	}

	bst.states = tree

	return nil
}

func (bst *LeveldbBlockStorage) marshal(i interface{}) ([]byte, error) {
	b, err := bst.st.defaultEnc.Encode(i)
	if err != nil {
		return nil, err
	}

	return storage.LeveldbDataWithEncoder(bst.st.defaultEnc, b), nil
}

func (bst *LeveldbBlockStorage) Commit() error {
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