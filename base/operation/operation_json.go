package operation

import (
	"encoding/json"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type baseOperationJSONPacker struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	FC OperationFact  `json:"fact"`
	FS []FactSign     `json:"fact_signs"`
}

func (bo BaseOperation) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(baseOperationJSONPacker{
		HintedHead: jsonenc.NewHintedHead(bo.Hint()),
		H:          bo.h,
		FC:         bo.fact,
		FS:         bo.fs,
	})
}

type baseOperationJSONUnpacker struct {
	jsonenc.HintedHead
	H  valuehash.Bytes   `json:"hash"`
	FC json.RawMessage   `json:"fact"`
	FS []json.RawMessage `json:"fact_signs"`
}

func (bo *BaseOperation) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ubo baseOperationJSONUnpacker
	if err := enc.Unmarshal(b, &ubo); err != nil {
		return err
	}

	fs := make([][]byte, len(ubo.FS))
	for i := range ubo.FS {
		fs[i] = ubo.FS[i]
	}

	return bo.unpack(enc, ubo.HintedHead.H, ubo.H, ubo.FC, fs)
}
