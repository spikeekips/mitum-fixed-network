package ballot

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type INITFactPackerJSON struct {
	P valuehash.Hash `json:"previous_block"`
}

func (fact INITFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		*BaseFactPackerJSON
		*INITFactPackerJSON
	}{
		BaseFactPackerJSON: fact.packerJSON(),
		INITFactPackerJSON: &INITFactPackerJSON{
			P: fact.previousBlock,
		},
	})
}

type INITFactUnpackerJSON struct {
	P valuehash.Bytes `json:"previous_block"`
}

func (fact *INITFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ufact struct {
		*BaseFactUnpackerJSON
		*INITFactUnpackerJSON
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

	fact.previousBlock = ufact.INITFactUnpackerJSON.P

	return nil
}
