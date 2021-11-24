package ballot // nolint:dupl

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type ACCEPTFactPackerJSON struct {
	P  valuehash.Hash `json:"proposal"`
	NB valuehash.Hash `json:"new_block"`
}

func (fact ACCEPTFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		*BaseFactPackerJSON
		*ACCEPTFactPackerJSON
	}{
		BaseFactPackerJSON: fact.packerJSON(),
		ACCEPTFactPackerJSON: &ACCEPTFactPackerJSON{
			P:  fact.proposal,
			NB: fact.newBlock,
		},
	})
}

type ACCEPTFactUnpackerJSON struct {
	P  valuehash.Bytes `json:"proposal"`
	NB valuehash.Bytes `json:"new_block"`
}

func (fact *ACCEPTFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ufact struct {
		*BaseFactUnpackerJSON
		*ACCEPTFactUnpackerJSON
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

	fact.proposal = ufact.ACCEPTFactUnpackerJSON.P
	fact.newBlock = ufact.ACCEPTFactUnpackerJSON.NB

	return nil
}
