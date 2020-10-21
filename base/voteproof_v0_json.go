package base

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/key"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type VoteproofV0FactJSONPacker struct {
	H valuehash.Hash
	F Fact
}

type VoteproofV0FactJSONUnpacker struct {
	H valuehash.Bytes
	F json.RawMessage
}

func (vv VoteproofV0FactJSONUnpacker) Hash() valuehash.Bytes {
	return vv.H
}

func (vv VoteproofV0FactJSONUnpacker) Fact() []byte {
	return vv.F
}

type VoteproofV0BallotJSONPacker struct {
	H valuehash.Hash
	A Address
}

type VoteproofV0BallotJSONUnpacker struct {
	H valuehash.Bytes
	A json.RawMessage
}

func (vv VoteproofV0BallotJSONUnpacker) Hash() valuehash.Bytes {
	return vv.H
}

func (vv VoteproofV0BallotJSONUnpacker) Address() []byte {
	return vv.A
}

type VoteproofV0PackJSON struct {
	jsonenc.HintedHead
	HT Height              `json:"height"`
	RD Round               `json:"round"`
	SS []Address           `json:"suffrages"`
	TH ThresholdRatio      `json:"threshold"`
	RS VoteResultType      `json:"result"`
	ST Stage               `json:"stage"`
	MJ Fact                `json:"majority"`
	FS []Fact              `json:"facts"`
	VS []VoteproofNodeFact `json:"votes"`
	FA localtime.Time      `json:"finished_at"`
	CL string              `json:"is_closed"`
}

func (vp VoteproofV0) MarshalJSON() ([]byte, error) {
	var isClosed string
	if vp.closed {
		isClosed = "true"
	} else {
		isClosed = "false"
	}

	return jsonenc.Marshal(VoteproofV0PackJSON{
		HintedHead: jsonenc.NewHintedHead(vp.Hint()),
		HT:         vp.height,
		RD:         vp.round,
		SS:         vp.suffrages,
		TH:         vp.thresholdRatio,
		RS:         vp.result,
		ST:         vp.stage,
		MJ:         vp.majority,
		FS:         vp.facts,
		VS:         vp.votes,
		FA:         localtime.NewTime(vp.finishedAt),
		CL:         isClosed,
	})
}

type VoteproofV0UnpackJSON struct {
	HT Height            `json:"height"`
	RD Round             `json:"round"`
	SS []AddressDecoder  `json:"suffrages"`
	TH ThresholdRatio    `json:"threshold"`
	RS VoteResultType    `json:"result"`
	ST Stage             `json:"stage"`
	MJ json.RawMessage   `json:"majority"`
	FS []json.RawMessage `json:"facts"`
	VS []json.RawMessage `json:"votes"`
	FA localtime.Time    `json:"finished_at"`
	CL string            `json:"is_closed"`
}

func (vp *VoteproofV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var vpp VoteproofV0UnpackJSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	bFacts := make([][]byte, len(vpp.FS))
	for i := range vpp.FS {
		bFacts[i] = vpp.FS[i]
	}

	vs := make([][]byte, len(vpp.VS))
	for i := range vpp.VS {
		vs[i] = vpp.VS[i]
	}

	return vp.unpack(
		enc,
		vpp.HT,
		vpp.RD,
		vpp.SS,
		vpp.TH,
		vpp.RS,
		vpp.ST,
		vpp.MJ,
		bFacts,
		vs,
		vpp.FA.Time,
		vpp.CL == "true",
	)
}

type VoteproofNodeFactPackJSON struct {
	AD Address        `json:"address"`
	BT valuehash.Hash `json:"ballot"`
	FC valuehash.Hash `json:"fact"`
	FS key.Signature  `json:"fact_signature"`
	SG key.Publickey  `json:"signer"`
}

func (vf VoteproofNodeFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(VoteproofNodeFactPackJSON{
		AD: vf.address,
		BT: vf.ballot,
		FC: vf.fact,
		FS: vf.factSignature,
		SG: vf.signer,
	})
}

type VoteproofNodeFactUnpackJSON struct {
	AD AddressDecoder       `json:"address"`
	BT valuehash.Bytes      `json:"ballot"`
	FC valuehash.Bytes      `json:"fact"`
	FS key.Signature        `json:"fact_signature"`
	SG key.PublickeyDecoder `json:"signer"`
}

func (vf *VoteproofNodeFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var vpp VoteproofNodeFactUnpackJSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	return vf.unpack(enc, vpp.AD, vpp.BT, vpp.FC, vpp.FS, vpp.SG)
}
