package network

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type HandoverSealV0JSONPack struct {
	*seal.BaseSealJSONPack
	AD base.Address `json:"address"`
	CI ConnInfo     `json:"conninfo"`
}

func (sl HandoverSealV0) MarshalJSON() ([]byte, error) {
	b := sl.BaseSeal.JSONPacker()

	return jsonenc.Marshal(HandoverSealV0JSONPack{
		BaseSealJSONPack: &b,
		AD:               sl.ad,
		CI:               sl.ci,
	})
}

type HandoverSealV0JSONUnpack struct {
	AD base.AddressDecoder `json:"address"`
	CI json.RawMessage     `json:"conninfo"`
}

func (sl *HandoverSealV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ub seal.BaseSeal
	if err := ub.UnpackJSON(b, enc); err != nil {
		return err
	}

	var usl HandoverSealV0JSONUnpack
	if err := enc.Unmarshal(b, &usl); err != nil {
		return err
	}

	return sl.unpack(enc, ub, usl.AD, usl.CI)
}
