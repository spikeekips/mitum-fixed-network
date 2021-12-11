package block

import (
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type ManifestV0 struct {
	hint.BaseHinter
	h              valuehash.Hash
	height         base.Height
	round          base.Round
	proposal       valuehash.Hash
	previousBlock  valuehash.Hash
	operationsHash valuehash.Hash
	statesHash     valuehash.Hash
	confirmedAt    time.Time
	createdAt      time.Time
}

func (bm ManifestV0) GenerateHash() valuehash.Hash {
	var operationsHashBytes []byte
	if bm.operationsHash != nil {
		operationsHashBytes = bm.operationsHash.Bytes()
	}

	var statesHashBytes []byte
	if bm.statesHash != nil {
		statesHashBytes = bm.statesHash.Bytes()
	}

	return valuehash.NewSHA256(util.ConcatBytesSlice(
		bm.height.Bytes(),
		bm.round.Bytes(),
		bm.proposal.Bytes(),
		bm.previousBlock.Bytes(),
		operationsHashBytes,
		statesHashBytes,
		localtime.NewTime(bm.confirmedAt).Bytes(),
		// NOTE createdAt does not included for Bytes(), because Bytes() is used
		// for Hash().
	))
}

func (bm ManifestV0) IsValid(networkID []byte) error {
	if bm.confirmedAt.IsZero() {
		return isvalid.InvalidError.Errorf("empty confirmedAt")
	}

	if err := isvalid.Check(networkID, false,
		bm.BaseHinter,
		bm.h,
		bm.height,
		bm.proposal,
		bm.previousBlock,
	); err != nil {
		return err
	}

	// NOTE operationsHash and statesHash are allowed to be empty.
	if err := isvalid.Check(networkID, true,
		bm.operationsHash,
		bm.statesHash,
	); err != nil && !errors.Is(err, valuehash.EmptyHashError) {
		return err
	}

	if !bm.h.Equal(bm.GenerateHash()) {
		return isvalid.InvalidError.Errorf("incorrect manifest hash")
	}

	return nil
}

func (bm ManifestV0) Hash() valuehash.Hash {
	return bm.h
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

func (bm ManifestV0) ConfirmedAt() time.Time {
	return bm.confirmedAt
}

func (bm ManifestV0) CreatedAt() time.Time {
	return bm.createdAt
}
