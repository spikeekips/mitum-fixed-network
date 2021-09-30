package operation

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/seal"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type BaseSealJSONPack struct {
	*seal.BaseSealJSONPack
	OPS []Operation `json:"operations"`
}

func (sl BaseSeal) MarshalJSON() ([]byte, error) {
	b := sl.BaseSeal.JSONPacker()

	return jsonenc.Marshal(BaseSealJSONPack{
		BaseSealJSONPack: &b,
		OPS:              sl.ops,
	})
}

type BaseSealJSONUnpack struct {
	OPS json.RawMessage `json:"operations"`
}

func (sl *BaseSeal) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ub seal.BaseSeal
	if err := ub.UnpackJSON(b, enc); err != nil {
		return err
	}

	var usl BaseSealJSONUnpack
	if err := enc.Unmarshal(b, &usl); err != nil {
		return err
	}

	return sl.unpack(enc, ub, usl.OPS)
}
