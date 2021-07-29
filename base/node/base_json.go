package node

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseV0PackerJSON struct {
	jsonenc.HintedHead
	AD base.Address  `json:"address"`
	PK key.Publickey `json:"publickey"`
}

func (bn BaseV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseV0PackerJSON{
		HintedHead: jsonenc.NewHintedHead(bn.Hint()),
		AD:         bn.address,
		PK:         bn.publickey,
	})
}

type BaseV0UnpackerJSON struct {
	AD base.AddressDecoder  `json:"address"`
	PK key.PublickeyDecoder `json:"publickey"`
}

func (bn *BaseV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var nbn BaseV0UnpackerJSON
	if err := enc.Unmarshal(b, &nbn); err != nil {
		return err
	}

	return bn.unpack(enc, nbn.AD, nbn.PK)
}
