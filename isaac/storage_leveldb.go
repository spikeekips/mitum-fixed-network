package isaac

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/syndtr/goleveldb/leveldb"
	leveldbutil "github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/xerrors"

	"github.com/spikeekips/avl"
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
	leveldbVoteProofHeightPrefix []byte = []byte{0x00, 0x02}
	leveldbSealPrefix            []byte = []byte{0x00, 0x03}
	leveldbProposalPrefix        []byte = []byte{0x00, 0x04}
	leveldbBlockOperationsPrefix []byte = []byte{0x00, 0x05}
	leveldbBlockStatesPrefix     []byte = []byte{0x00, 0x06}
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

func (st *LeveldbStorage) LastBlock() (Block, error) {
	var raw []byte

	iter := st.db.NewIterator(leveldbutil.BytesPrefix(leveldbBlockHeightPrefix), nil)
	if iter.Last() {
		raw = util.CopyBytes(iter.Value())
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return nil, err
	}

	if raw == nil {
		return nil, nil
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

	return hinter.(Block), nil
}

func (st *LeveldbStorage) loadLastVoteProof(stage Stage) (VoteProof, error) {
	return st.filterVoteProof(leveldbVoteProofHeightPrefix, stage)
}

func (st *LeveldbStorage) newVoteProof(voteProof VoteProof) error {
	st.Log().Debug().
		Int64("height", voteProof.Height().Int64()).
		Uint64("round", voteProof.Round().Uint64()).
		Str("stage", voteProof.Stage().String()).
		Msg("voteproof stored")

	raw, err := st.defaultEnc.Marshal(voteProof)
	if err != nil {
		return err
	}

	hb := storage.LeveldbDataWithEncoder(st.defaultEnc, raw)
	return st.db.Put(leveldbVoteProofKey(voteProof), hb, nil)
}

func (st *LeveldbStorage) LastINITVoteProof() (VoteProof, error) {
	return st.loadLastVoteProof(StageINIT)
}

func (st *LeveldbStorage) NewINITVoteProof(voteProof VoteProof) error {
	return st.newVoteProof(voteProof)
}

func (st *LeveldbStorage) filterVoteProof(prefix []byte, stage Stage) (VoteProof, error) {
	var raw []byte

	iter := st.db.NewIterator(leveldbutil.BytesPrefix(prefix), nil)
	if iter.Last() {
		for {
			key := util.CopyBytes(iter.Key())

			var height int64
			var round uint64
			var stg uint8
			n, err := fmt.Sscanf(
				string(key[len(leveldbVoteProofHeightPrefix):]),
				"%020d-%020d-%d", &height, &round, &stg,
			)
			if err != nil {
				return nil, err
			}

			if n != 3 {
				return nil, xerrors.Errorf("invalid formatted key found: key=%q", string(key))
			}

			if Stage(stg) != stage {
				if !iter.Prev() {
					break
				}
				continue
			}

			raw = util.CopyBytes(iter.Value())

			break
		}
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return nil, err
	}

	if raw == nil {
		return nil, nil
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

	return hinter.(VoteProof), nil
}

func (st *LeveldbStorage) LastINITVoteProofOfHeight(height Height) (VoteProof, error) {
	return st.filterVoteProof(leveldbVoteProofKeyByHeight(height), StageINIT)
}

func (st *LeveldbStorage) LastACCEPTVoteProofOfHeight(height Height) (VoteProof, error) {
	return st.filterVoteProof(leveldbVoteProofKeyByHeight(height), StageACCEPT)
}

func (st *LeveldbStorage) LastACCEPTVoteProof() (VoteProof, error) {
	return st.loadLastVoteProof(StageACCEPT)
}

func (st *LeveldbStorage) NewACCEPTVoteProof(voteProof VoteProof) error {
	return st.newVoteProof(voteProof)
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
	raw, err := st.defaultEnc.Marshal(sl)
	if err != nil {
		return err
	}

	hb := storage.LeveldbDataWithEncoder(st.defaultEnc, raw)

	return st.db.Put(st.sealKey(sl.Hash()), hb, nil)
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
		bst.batch.Put(leveldbBlockKey(block), b)
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
	b, err := bst.st.defaultEnc.Marshal(i)
	if err != nil {
		return nil, err
	}

	return storage.LeveldbDataWithEncoder(bst.st.defaultEnc, b), nil
}

func (bst *LeveldbBlockStorage) Commit() error {
	return bst.st.db.Write(bst.batch, nil)
}

func leveldbBlockKey(block Block) []byte {
	return util.ConcatSlice([][]byte{
		leveldbBlockHeightPrefix,
		[]byte(fmt.Sprintf("%020d", block.Height().Int64())),
	})
}

func leveldbVoteProofKey(voteProof VoteProof) []byte {
	return util.ConcatSlice([][]byte{
		leveldbVoteProofHeightPrefix,
		[]byte(fmt.Sprintf(
			"%020d-%020d-%d",
			voteProof.Height().Int64(),
			voteProof.Round().Uint64(),
			voteProof.Stage(),
		)),
	})
}

func leveldbVoteProofKeyByHeight(height Height) []byte {
	return util.ConcatSlice([][]byte{
		leveldbVoteProofHeightPrefix,
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
