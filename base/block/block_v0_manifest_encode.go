package block

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (bm *ManifestV0) unpack(
	_ encoder.Encoder,
	h valuehash.Hash,
	height base.Height,
	round base.Round,
	proposal,
	previousBlock,
	operationsHash,
	statesHash valuehash.Hash,
	confirmedAt time.Time,
	createdAt time.Time,
) error {
	if operationsHash != nil && operationsHash.Empty() {
		operationsHash = nil
	}

	if statesHash != nil && statesHash.Empty() {
		statesHash = nil
	}

	bm.h = h
	bm.height = height
	bm.round = round
	bm.proposal = proposal
	bm.previousBlock = previousBlock
	bm.operationsHash = operationsHash
	bm.statesHash = statesHash
	bm.confirmedAt = confirmedAt
	bm.createdAt = createdAt

	return nil
}
