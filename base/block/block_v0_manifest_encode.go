package block

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
)

func (bm *ManifestV0) unpack(
	enc encoder.Encoder,
	bHash []byte,
	height base.Height,
	round base.Round,
	bProposal,
	bPreviousBlock,
	bBlockOperations,
	bBlockStates []byte,
	createdAt time.Time,
) error {
	var h, pr, pb, bo, bs valuehash.Hash
	var err error
	if h, err = valuehash.Decode(enc, bHash); err != nil {
		return err
	}
	if pr, err = valuehash.Decode(enc, bProposal); err != nil {
		return err
	}
	if pb, err = valuehash.Decode(enc, bPreviousBlock); err != nil {
		return err
	}
	if bBlockOperations != nil {
		if bo, err = valuehash.Decode(enc, bBlockOperations); err != nil {
			return err
		}
	}

	if bBlockStates != nil {
		if bs, err = valuehash.Decode(enc, bBlockStates); err != nil {
			return err
		}
	}

	bm.h = h
	bm.height = height
	bm.round = round
	bm.proposal = pr
	bm.previousBlock = pb
	bm.operationsHash = bo
	bm.statesHash = bs
	bm.createdAt = createdAt

	return nil
}
