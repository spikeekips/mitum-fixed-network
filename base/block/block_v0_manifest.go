package block

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

type ManifestV0 struct {
	h              valuehash.Hash
	height         base.Height
	round          base.Round
	proposal       valuehash.Hash
	previousBlock  valuehash.Hash
	operationsHash valuehash.Hash
	statesHash     valuehash.Hash
	createdAt      time.Time
}

func (bm ManifestV0) GenerateHash() (valuehash.Hash, error) {
	return valuehash.NewSHA256(bm.Bytes()), nil
}

func (bm ManifestV0) IsValid([]byte) error {
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

func (bm ManifestV0) Hint() hint.Hint {
	return ManifestV0Hint
}

func (bm ManifestV0) Hash() valuehash.Hash {
	return bm.h
}

func (bm ManifestV0) Bytes() []byte {
	var operationsHashBytes []byte
	if bm.operationsHash != nil {
		operationsHashBytes = bm.operationsHash.Bytes()
	}

	var statesHashBytes []byte
	if bm.statesHash != nil {
		statesHashBytes = bm.statesHash.Bytes()
	}

	return util.ConcatBytesSlice(
		bm.height.Bytes(),
		bm.round.Bytes(),
		bm.proposal.Bytes(),
		bm.previousBlock.Bytes(),
		operationsHashBytes,
		statesHashBytes,
		// NOTE createdAt does not included for Bytes(), because Bytes() is used
		// for Hash().
	)
}

func (bm ManifestV0) Height() base.Height {
	return bm.height
}

func (bm ManifestV0) Round() base.Round {
	return bm.round
}

func (bm ManifestV0) Proposal() valuehash.Hash {
	return bm.proposal
}

func (bm ManifestV0) PreviousBlock() valuehash.Hash {
	return bm.previousBlock
}

func (bm ManifestV0) OperationsHash() valuehash.Hash {
	return bm.operationsHash
}

func (bm ManifestV0) StatesHash() valuehash.Hash {
	return bm.statesHash
}

func (bm ManifestV0) CreatedAt() time.Time {
	return bm.createdAt
}
