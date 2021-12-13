package ballot

import (
	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type ProposalFactPackerJSON struct {
	PR  base.Address     `json:"proposer"`
	OPS []valuehash.Hash `json:"operations"`
	PA  localtime.Time   `json:"proposed_at"`
}

func (fact ProposalFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		*BaseFactPackerJSON
		*ProposalFactPackerJSON
	}{
		BaseFactPackerJSON: fact.packerJSON(),
		ProposalFactPackerJSON: &ProposalFactPackerJSON{
			PR:  fact.proposer,
			OPS: fact.ops,
			PA:  localtime.NewTime(fact.proposedAt),
		},
	})
}

type ProposalFactUnpackerJSON struct {
	PR  base.AddressDecoder `json:"proposer"`
	OPS []valuehash.Bytes   `json:"operations"`
	PA  localtime.Time      `json:"proposed_at"`
}

func (fact *ProposalFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ufact struct {
		*BaseFactUnpackerJSON
		*ProposalFactUnpackerJSON
	}

	if err := enc.Unmarshal(b, &ufact); err != nil {
		return err
	}

	if err := fact.BaseFact.unpack(
		enc,
		ufact.BaseFactUnpackerJSON.HI,
		ufact.BaseFactUnpackerJSON.H,
		ufact.BaseFactUnpackerJSON.HT,
		ufact.BaseFactUnpackerJSON.R,
	); err != nil {
		return err
	}

	pr, err := ufact.ProposalFactUnpackerJSON.PR.Encode(enc)
	if err != nil {
		return err
	}
	fact.proposer = pr

	ops := make([]valuehash.Hash, len(ufact.ProposalFactUnpackerJSON.OPS))
	for i := range ufact.ProposalFactUnpackerJSON.OPS {
		ops[i] = ufact.ProposalFactUnpackerJSON.OPS[i]
	}

	fact.ops = ops
	fact.proposedAt = ufact.ProposalFactUnpackerJSON.PA.Time

	return nil
}
