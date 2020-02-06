package mitum

import (
	"time"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
	"golang.org/x/xerrors"
)

var BlockV0Hint hint.Hint = hint.MustHint(BlockType, "0.1")

type BlockV0 struct {
	h               valuehash.Hash
	height          Height
	round           Round
	proposal        valuehash.Hash
	previousBlock   valuehash.Hash
	blockOperations valuehash.Hash
	blockStates     valuehash.Hash
	initVoteProof   VoteProof
	acceptVoteProof VoteProof
	createdAt       time.Time
}

func NewBlockV0(
	height Height,
	round Round,
	proposal valuehash.Hash,
	previousBlock valuehash.Hash,
	blockOperations valuehash.Hash,
	blockStates valuehash.Hash,
	initVoteProof VoteProof,
	acceptVoteProof VoteProof,
	b []byte,
) (BlockV0, error) {
	root, err := GenerateBlockV0Hash(
		height,
		round,
		proposal,
		previousBlock,
		blockOperations,
		blockStates,
		b,
	)
	if err != nil {
		return BlockV0{}, err
	}

	return BlockV0{
		h:               root,
		previousBlock:   previousBlock,
		height:          height,
		round:           round,
		proposal:        proposal,
		initVoteProof:   initVoteProof,
		acceptVoteProof: acceptVoteProof,
		blockOperations: blockOperations,
		blockStates:     blockStates,
		createdAt:       localtime.Now(),
	}, nil
}

func GenerateBlockV0Hash(
	height Height,
	round Round,
	proposal valuehash.Hash,
	previousBlock valuehash.Hash,
	blockOperations valuehash.Hash,
	blockStates valuehash.Hash,
	b []byte,
) (valuehash.Hash, error) {
	e := util.ConcatSlice([][]byte{
		height.Bytes(),
		round.Bytes(),
		proposal.Bytes(),
		previousBlock.Bytes(),
		blockOperations.Bytes(),
		blockStates.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (bm BlockV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		bm.h,
		bm.height,
		bm.proposal,
		bm.previousBlock,
	}, b); err != nil {
		return err
	}

	// NOTE blockOperations and blockStates are allowed to be empty.
	if err := isvalid.Check([]isvalid.IsValider{
		bm.blockOperations,
		bm.blockStates,
	}, b); err != nil && !xerrors.Is(err, valuehash.EmptyHashError) {
		return err
	}

	gh, err := GenerateBlockV0Hash(
		bm.height,
		bm.round,
		bm.proposal,
		bm.previousBlock,
		bm.blockOperations,
		bm.blockStates,
		b,
	)
	if err != nil {
		return err
	} else if !bm.h.Equal(gh) {
		return xerrors.Errorf("incorrect hash; hash=%s != generated=%s", bm.h, gh)
	}

	return nil
}

func (bm BlockV0) Hint() hint.Hint {
	return BlockV0Hint
}

func (bm BlockV0) Bytes() []byte {
	return nil
}

func (bm BlockV0) Hash() valuehash.Hash {
	return bm.h
}

func (bm BlockV0) Height() Height {
	return bm.height
}

func (bm BlockV0) Round() Round {
	return bm.round
}

func (bm BlockV0) Proposal() valuehash.Hash {
	return bm.proposal
}

func (bm BlockV0) PreviousBlock() valuehash.Hash {
	return bm.previousBlock
}

func (bm BlockV0) Operations() valuehash.Hash {
	return bm.blockOperations
}

func (bm BlockV0) States() valuehash.Hash {
	return bm.blockStates
}

func (bm BlockV0) INITVoteProof() VoteProof {
	return bm.initVoteProof
}

func (bm BlockV0) ACCEPTVoteProof() VoteProof {
	return bm.acceptVoteProof
}

func (bm BlockV0) CreatedAt() time.Time {
	return bm.createdAt
}
