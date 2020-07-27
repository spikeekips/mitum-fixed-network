package policy

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/operation"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type SetPolicyV0PackerJSON struct {
	jsonenc.HintedHead

	HS valuehash.Hash       `json:"hash"`
	FS []operation.FactSign `json:"fact_signs"`
	TK []byte               `json:"token"`
	PO PolicyV0             `json:"policy"`
}

func (spo SetPolicyV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(SetPolicyV0PackerJSON{
		HintedHead: jsonenc.NewHintedHead(spo.Hint()),
		HS:         spo.h,
		FS:         spo.fs,
		TK:         spo.token,
		PO:         spo.SetPolicyFactV0.PolicyV0,
	})
}

type SetPolicyV0UnpackJSON struct {
	H  valuehash.Bytes   `json:"hash"`
	FS []json.RawMessage `json:"fact_signs"`
	TK []byte            `json:"token"`
	PO json.RawMessage   `json:"policy"`
}

func (spo *SetPolicyV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uspo SetPolicyV0UnpackJSON
	if err := enc.Unmarshal(b, &uspo); err != nil {
		return err
	}

	fs := make([][]byte, len(uspo.FS))
	for i := range uspo.FS {
		fs[i] = uspo.FS[i]
	}

	return spo.unpack(enc, uspo.H, fs, uspo.TK, uspo.PO)
}
