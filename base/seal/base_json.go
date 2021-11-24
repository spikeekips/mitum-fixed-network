package seal

import (
	"github.com/spikeekips/mitum/base/key"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BaseSealJSONPack struct {
	jsonenc.HintedHead
	HH valuehash.Hash `json:"hash"`
	BH valuehash.Hash `json:"body_hash"`
	SN key.Publickey  `json:"signer"`
	SG key.Signature  `json:"signature"`
	SA localtime.Time `json:"signed_at"`
}

func (sl BaseSeal) JSONPacker() *BaseSealJSONPack {
	return &BaseSealJSONPack{
		HintedHead: jsonenc.NewHintedHead(sl.Hint()),
		HH:         sl.h,
		BH:         sl.bodyHash,
		SN:         sl.signer,
		SG:         sl.signature,
		SA:         localtime.NewTime(sl.signedAt),
	}
}

type BaseSealJSONUnpack struct {
	HH valuehash.Bytes      `json:"hash"`
	BH valuehash.Bytes      `json:"body_hash"`
	SN key.PublickeyDecoder `json:"signer"`
	SG key.Signature        `json:"signature"`
	SA localtime.Time       `json:"signed_at"`
}

func (sl *BaseSeal) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uht jsonenc.HintedHead
	if err := enc.Unmarshal(b, &uht); err != nil {
		return err
	}

	var usl BaseSealJSONUnpack
	if err := enc.Unmarshal(b, &usl); err != nil {
		return err
	}

	return sl.unpack(enc, uht.H, usl.HH, usl.BH, usl.SN, usl.SG, usl.SA.Time)
}
