package operation

import (
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
)

type BaseFactSignJSONPacker struct {
	jsonenc.HintedHead
	SN key.Publickey      `json:"signer"`
	SG key.Signature      `json:"signature"`
	SA localtime.JSONTime `json:"signed_at"`
}

func (fs BaseFactSign) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseFactSignJSONPacker{
		HintedHead: jsonenc.NewHintedHead(fs.Hint()),
		SN:         fs.signer,
		SG:         fs.signature,
		SA:         localtime.NewJSONTime(fs.signedAt),
	})
}

type BaseFactSignJSONUnpacker struct {
	SN encoder.HintedString `json:"signer"`
	SG key.Signature        `json:"signature"`
	SA localtime.JSONTime   `json:"signed_at"`
}

func (fs *BaseFactSign) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ufs BaseFactSignJSONUnpacker
	if err := enc.Unmarshal(b, &ufs); err != nil {
		return err
	}

	return fs.unpack(enc, ufs.SN, ufs.SG, ufs.SA.Time)
}
