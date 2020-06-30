package operation

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/key"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type SealJSONPack struct {
	jsonenc.HintedHead
	H   valuehash.Hash     `json:"hash"`
	BH  valuehash.Hash     `json:"body_hash"`
	SN  key.Publickey      `json:"signer"`
	SG  key.Signature      `json:"signature"`
	SA  localtime.JSONTime `json:"signed_at"`
	OPS []Operation        `json:"operations"`
}

func (sl Seal) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(SealJSONPack{
		HintedHead: jsonenc.NewHintedHead(sl.Hint()),
		H:          sl.h,
		BH:         sl.bodyHash,
		SN:         sl.signer,
		SG:         sl.signature,
		SA:         localtime.NewJSONTime(sl.signedAt),
		OPS:        sl.ops,
	})
}

type SealJSONUnpack struct {
	H   valuehash.Bytes    `json:"hash"`
	BH  valuehash.Bytes    `json:"body_hash"`
	SN  json.RawMessage    `json:"signer"`
	SG  key.Signature      `json:"signature"`
	SA  localtime.JSONTime `json:"signed_at"`
	OPS []json.RawMessage  `json:"operations"`
}

func (sl *Seal) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var usl SealJSONUnpack
	if err := enc.Unmarshal(b, &usl); err != nil {
		return err
	}

	ops := make([][]byte, len(usl.OPS))
	for i, b := range usl.OPS {
		ops[i] = b
	}

	return sl.unpack(enc, usl.H, usl.BH, usl.SN, usl.SG, usl.SA.Time, ops)
}
