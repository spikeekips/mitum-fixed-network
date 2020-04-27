package operation

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
)

type SealJSONPack struct {
	jsonencoder.HintedHead
	H   valuehash.Hash     `json:"hash"`
	BH  valuehash.Hash     `json:"body_hash"`
	SN  key.Publickey      `json:"signer"`
	SG  key.Signature      `json:"signature"`
	SA  localtime.JSONTime `json:"signed_at"`
	OPS []Operation        `json:"operations"`
}

func (sl Seal) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(SealJSONPack{
		HintedHead: jsonencoder.NewHintedHead(sl.Hint()),
		H:          sl.h,
		BH:         sl.bodyHash,
		SN:         sl.signer,
		SG:         sl.signature,
		SA:         localtime.NewJSONTime(sl.signedAt),
		OPS:        sl.ops,
	})
}

type SealJSONUnpack struct {
	H   json.RawMessage    `json:"hash"`
	BH  json.RawMessage    `json:"body_hash"`
	SN  json.RawMessage    `json:"signer"`
	SG  key.Signature      `json:"signature"`
	SA  localtime.JSONTime `json:"signed_at"`
	OPS []json.RawMessage  `json:"operations"`
}

func (sl *Seal) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
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
