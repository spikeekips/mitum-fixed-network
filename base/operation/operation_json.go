package operation

import (
	"encoding/json"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (bo BaseOperation) JSONM() map[string]interface{} {
	return map[string]interface{}{
		"_hint":      bo.Hint(),
		"hash":       bo.h,
		"fact":       bo.fact,
		"fact_signs": bo.fs,
	}
}

func (bo BaseOperation) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(bo.JSONM())
}

type baseOperationJSONUnpacker struct {
	H  valuehash.Bytes `json:"hash"`
	FC json.RawMessage `json:"fact"`
	FS json.RawMessage `json:"fact_signs"`
}

func (bo *BaseOperation) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo baseOperationJSONUnpacker
	if err := enc.Unmarshal(b, &ubo); err != nil {
		return err
	}

	return bo.unpack(enc, ubo.H, ubo.FC, ubo.FS)
}
