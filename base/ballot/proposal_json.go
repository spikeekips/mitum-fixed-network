package ballot

import (
	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type ProposalFactPackerJSON struct {
	PR base.Address     `json:"proposer"`
	SL []valuehash.Hash `json:"seals"`
	PA localtime.Time   `json:"proposed_at"`
}

func (fact ProposalFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		*BaseFactPackerJSON
		*ProposalFactPackerJSON
	}{
		BaseFactPackerJSON: fact.packerJSON(),
		ProposalFactPackerJSON: &ProposalFactPackerJSON{
			PR: fact.proposer,
			SL: fact.seals,
			PA: localtime.NewTime(fact.proposedAt),
		},
	})
}

type ProposalFactUnpackerJSON struct {
	PR base.AddressDecoder `json:"proposer"`
	SL []valuehash.Bytes   `json:"seals"`
	PA localtime.Time      `json:"proposed_at"`
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

	sl := make([]valuehash.Hash, len(ufact.ProposalFactUnpackerJSON.SL))
	for i := range ufact.ProposalFactUnpackerJSON.SL {
		sl[i] = ufact.ProposalFactUnpackerJSON.SL[i]
	}

	fact.seals = sl
	fact.proposedAt = ufact.ProposalFactUnpackerJSON.PA.Time

	return nil
}
