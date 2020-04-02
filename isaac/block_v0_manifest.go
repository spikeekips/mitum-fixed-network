package isaac

import (
	"time"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
	"golang.org/x/xerrors"
)

type BlockManifestV0 struct {
	h              valuehash.Hash
	height         Height
	round          Round
	proposal       valuehash.Hash
	previousBlock  valuehash.Hash
	operationsHash valuehash.Hash
	statesHash     valuehash.Hash
	createdAt      time.Time
}

func (bm BlockManifestV0) GenerateHash() (valuehash.Hash, error) {
	return valuehash.NewSHA256(bm.Bytes()), nil
}

func (bm BlockManifestV0) IsValid([]byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		bm.h,
		bm.height,
		bm.proposal,
		bm.previousBlock,
	}, nil, false); err != nil {
		return err
	}

	// NOTE operationsHash and statesHash are allowed to be empty.
	if err := isvalid.Check([]isvalid.IsValider{
		bm.operationsHash,
		bm.statesHash,
	}, nil, true); err != nil && !xerrors.Is(err, valuehash.EmptyHashError) {
		return err
	}

	if h, err := bm.GenerateHash(); err != nil {
		return err
	} else if !bm.h.Equal(h) {
		return xerrors.Errorf("incorrect hash; hash=%s != generated=%s", bm.h, h)
	}

	return nil
}

func (bm BlockManifestV0) Hint() hint.Hint {
	return BlockManifestV0Hint
}

func (bm BlockManifestV0) Hash() valuehash.Hash {
	return bm.h
}

func (bm BlockManifestV0) Bytes() []byte {
	var operationsHashBytes []byte
	if bm.operationsHash != nil {
		operationsHashBytes = bm.operationsHash.Bytes()
	}

	var statesHashBytes []byte
	if bm.statesHash != nil {
		statesHashBytes = bm.statesHash.Bytes()
	}

	return util.ConcatSlice([][]byte{
		bm.height.Bytes(),
		bm.round.Bytes(),
		bm.proposal.Bytes(),
		bm.previousBlock.Bytes(),
		operationsHashBytes,
		statesHashBytes,
		// NOTE createdAt does not included for Bytes(), because Bytes() is used
		// for Hash().
	})
}

func (bm BlockManifestV0) Height() Height {
	return bm.height
}

func (bm BlockManifestV0) Round() Round {
	return bm.round
}

func (bm BlockManifestV0) Proposal() valuehash.Hash {
	return bm.proposal
}

func (bm BlockManifestV0) PreviousBlock() valuehash.Hash {
	return bm.previousBlock
}

func (bm BlockManifestV0) OperationsHash() valuehash.Hash {
	return bm.operationsHash
}

func (bm BlockManifestV0) StatesHash() valuehash.Hash {
	return bm.statesHash
}

func (bm BlockManifestV0) CreatedAt() time.Time {
	return bm.createdAt
}
