package base

import (
	"github.com/spikeekips/mitum/base/key"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseNodeV0PackerJSON struct {
	jsonenc.HintedHead
	AD Address       `json:"address"`
	PK key.Publickey `json:"publickey"`
}

func (bn BaseNodeV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseNodeV0PackerJSON{
		HintedHead: jsonenc.NewHintedHead(bn.Hint()),
		AD:         bn.address,
		PK:         bn.publickey,
	})
}

type BaseNodeV0UnpackerJSON struct {
	AD AddressDecoder       `json:"address"`
	PK key.PublickeyDecoder `json:"publickey"`
}

func (bn *BaseNodeV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var nbn BaseNodeV0UnpackerJSON
	if err := enc.Unmarshal(b, &nbn); err != nil {
		return err
	}

	return bn.unpack(enc, nbn.AD, nbn.PK)
}
