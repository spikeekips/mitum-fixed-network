package ballot

import (
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
)

func (pr *ProposalV0) unpack(
	enc encoder.Encoder,
	bb BaseBallotV0,
	bf BaseBallotFactV0,
	bOperations,
	bSeals [][]byte,
) error {
	var ol, sl []valuehash.Hash
	for _, r := range bOperations {
		if h, err := valuehash.Decode(enc, r); err != nil {
			return err
		} else {
			ol = append(ol, h)
		}
	}

	for _, r := range bSeals {
		if h, err := valuehash.Decode(enc, r); err != nil {
			return err
		} else {
			sl = append(sl, h)
		}
	}

	pr.BaseBallotV0 = bb
	pr.ProposalFactV0 = ProposalFactV0{
		BaseBallotFactV0: bf,
		operations:       ol,
		seals:            sl,
	}

	return nil
}
