package base

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
)

type VoteproofV0PackJSON struct {
	jsonencoder.HintedHead
	HT Height             `json:"height"`
	RD Round              `json:"round"`
	TH Threshold          `json:"threshold"`
	RS VoteResultType     `json:"result"`
	ST Stage              `json:"stage"`
	MJ Fact               `json:"majority"`
	FS [][2]interface{}   `json:"facts"`
	BS [][2]interface{}   `json:"ballots"`
	VS [][2]interface{}   `json:"votes"`
	FA localtime.JSONTime `json:"finished_at"`
	CL string             `json:"is_closed"`
}

func (vp VoteproofV0) MarshalJSON() ([]byte, error) {
	var facts [][2]interface{} // nolint
	for h, f := range vp.facts {
		facts = append(facts, [2]interface{}{h, f})
	}

	var ballots [][2]interface{} // nolint
	for a, h := range vp.ballots {
		ballots = append(ballots, [2]interface{}{a, h})
	}

	var votes [][2]interface{} // nolint
	for a := range vp.votes {
		votes = append(votes, [2]interface{}{a, vp.votes[a]})
	}

	var isClosed string
	if vp.closed {
		isClosed = "true"
	} else {
		isClosed = "false"
	}

	return jsonencoder.Marshal(VoteproofV0PackJSON{
		HintedHead: jsonencoder.NewHintedHead(vp.Hint()),
		HT:         vp.height,
		RD:         vp.round,
		TH:         vp.threshold,
		RS:         vp.result,
		ST:         vp.stage,
		MJ:         vp.majority,
		FS:         facts,
		BS:         ballots,
		VS:         votes,
		FA:         localtime.NewJSONTime(vp.finishedAt),
		CL:         isClosed,
	})
}

type VoteproofV0UnpackJSON struct {
	HT Height               `json:"height"`
	RD Round                `json:"round"`
	TH Threshold            `json:"threshold"`
	RS VoteResultType       `json:"result"`
	ST Stage                `json:"stage"`
	MJ json.RawMessage      `json:"majority"`
	FS [][2]json.RawMessage `json:"facts"`
	BS [][2]json.RawMessage `json:"ballots"`
	VS [][2]json.RawMessage `json:"votes"`
	FA localtime.JSONTime   `json:"finished_at"`
	CL string               `json:"is_closed"`
}

func (vp *VoteproofV0) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error { // nolint
	var vpp VoteproofV0UnpackJSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	fs := make([][2][]byte, len(vpp.FS))
	for i := range vpp.FS {
		r := vpp.FS[i]
		fs[i] = [2][]byte{r[0], r[1]}
	}

	bs := make([][2][]byte, len(vpp.BS))
	for i := range vpp.BS {
		r := vpp.BS[i]
		bs[i] = [2][]byte{r[0], r[1]}
	}

	vs := make([][2][]byte, len(vpp.VS))
	for i := range vpp.VS {
		r := vpp.VS[i]
		vs[i] = [2][]byte{r[0], r[1]}
	}

	return vp.unpack(
		enc,
		vpp.HT,
		vpp.RD,
		vpp.TH,
		vpp.RS,
		vpp.ST,
		vpp.MJ,
		fs,
		bs,
		vs,
		vpp.FA.Time,
		vpp.CL == "true",
	)
}

type VoteproofNodeFactPackJSON struct {
	FC valuehash.Hash `json:"fact"`
	FS key.Signature  `json:"fact_signature"`
	SG key.Publickey  `json:"signer"`
}

func (vf VoteproofNodeFact) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(VoteproofNodeFactPackJSON{
		FC: vf.fact,
		FS: vf.factSignature,
		SG: vf.signer,
	})
}

type VoteproofNodeFactUnpackJSON struct {
	FC json.RawMessage `json:"fact"`
	FS key.Signature   `json:"fact_signature"`
	SG json.RawMessage `json:"signer"`
}

func (vf *VoteproofNodeFact) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
	var vpp VoteproofNodeFactUnpackJSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	return vf.unpack(enc, vpp.FC, vpp.FS, vpp.SG)
}
