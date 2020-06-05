package base

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/key"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseNodeV0PackerJSON struct {
	jsonencoder.HintedHead
	AD Address       `json:"address"`
	PK key.Publickey `json:"publickey"`
}

func (bn BaseNodeV0) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(BaseNodeV0PackerJSON{
		HintedHead: jsonencoder.NewHintedHead(bn.Hint()),
		AD:         bn.address,
		PK:         bn.publickey,
	})
}

type BaseNodeV0UnpackerJSON struct {
	AD json.RawMessage `json:"address"`
	PK json.RawMessage `json:"publickey"`
}

func (bn *BaseNodeV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	var nbn BaseNodeV0UnpackerJSON
	if err := enc.Unmarshal(b, &nbn); err != nil {
		return err
	}

	return bn.unpack(enc, nbn.AD, nbn.PK)
}
