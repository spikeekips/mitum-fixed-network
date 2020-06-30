package ballot

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (pr *ProposalV0) unpack(
	_ encoder.Encoder,
	bb BaseBallotV0,
	bf BaseBallotFactV0,
	ops,
	sls []valuehash.Hash,
) error {
	pr.BaseBallotV0 = bb
	pr.ProposalFactV0 = ProposalFactV0{
		BaseBallotFactV0: bf,
		operations:       ops,
		seals:            sls,
	}

	return nil
}
