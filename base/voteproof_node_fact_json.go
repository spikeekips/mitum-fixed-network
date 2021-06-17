package base

import (
	"github.com/spikeekips/mitum/base/key"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BaseVoteproofNodeFactPackJSON struct {
	jsonenc.HintedHead
	AD Address        `json:"address"`
	BT valuehash.Hash `json:"ballot"`
	FC valuehash.Hash `json:"fact"`
	FS key.Signature  `json:"fact_signature"`
	SG key.Publickey  `json:"signer"`
}

func (vf BaseVoteproofNodeFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BaseVoteproofNodeFactPackJSON{
		HintedHead: jsonenc.NewHintedHead(vf.Hint()),
		AD:         vf.address,
		BT:         vf.ballot,
		FC:         vf.fact,
		FS:         vf.factSignature,
		SG:         vf.signer,
	})
}

type BaseVoteproofNodeFactUnpackJSON struct {
	AD AddressDecoder       `json:"address"`
	BT valuehash.Bytes      `json:"ballot"`
	FC valuehash.Bytes      `json:"fact"`
	FS key.Signature        `json:"fact_signature"`
	SG key.PublickeyDecoder `json:"signer"`
}

func (vf *BaseVoteproofNodeFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var vpp BaseVoteproofNodeFactUnpackJSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	return vf.unpack(enc, vpp.AD, vpp.BT, vpp.FC, vpp.FS, vpp.SG)
}
