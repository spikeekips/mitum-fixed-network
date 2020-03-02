package operation

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type SealJSONPack struct {
	encoder.JSONPackHintedHead
	H   valuehash.Hash     `json:"hash"`
	BH  valuehash.Hash     `json:"body_hash"`
	SN  key.Publickey      `json:"signer"`
	SG  key.Signature      `json:"signature"`
	SA  localtime.JSONTime `json:"signed_at"`
	OPS []Operation        `json:"operations"`
}

func (sl Seal) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(SealJSONPack{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(sl.Hint()),
		H:                  sl.h,
		BH:                 sl.bodyHash,
		SN:                 sl.signer,
		SG:                 sl.signature,
		SA:                 localtime.NewJSONTime(sl.signedAt),
		OPS:                sl.ops,
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

func (sl *Seal) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var usl SealJSONUnpack
	if err := enc.Unmarshal(b, &usl); err != nil {
		return err
	}

	var err error
	var h, bodyHash valuehash.Hash
	if h, err = valuehash.Decode(enc, usl.H); err != nil {
		return err
	}
	if bodyHash, err = valuehash.Decode(enc, usl.BH); err != nil {
		return err
	}

	var signer key.Publickey
	if signer, err = key.DecodePublickey(enc, usl.SN); err != nil {
		return err
	}

	var ops []Operation
	for _, r := range usl.OPS {
		if op, err := DecodeOperation(enc, r); err != nil {
			return err
		} else {
			ops = append(ops, op)
		}
	}

	sl.h = h
	sl.bodyHash = bodyHash
	sl.signer = signer
	sl.signature = usl.SG
	sl.signedAt = usl.SA.Time
	sl.ops = ops

	return nil
}
