package ballot

import (
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
)

func (sb *SIGNBallotV0) unpack(
	enc encoder.Encoder,
	bb BaseBallotV0,
	bf BaseBallotFactV0,
	bProposal,
	bNewBlock []byte,
) error {
	var epr, enb valuehash.Hash
	if i, err := valuehash.Decode(enc, bProposal); err != nil {
		return err
	} else {
		epr = i
	}

	if i, err := valuehash.Decode(enc, bNewBlock); err != nil {
		return err
	} else {
		enb = i
	}

	sb.BaseBallotV0 = bb
	sb.SIGNBallotFactV0 = SIGNBallotFactV0{
		BaseBallotFactV0: bf,
		proposal:         epr,
		newBlock:         enb,
	}

	return nil
}
