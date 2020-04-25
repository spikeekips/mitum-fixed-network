package base

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
)

func (vp VoteproofV0) MarshalBSON() ([]byte, error) {
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

	m := bson.M{
		"height":      vp.height,
		"round":       vp.round,
		"threshold":   vp.threshold,
		"result":      vp.result,
		"stage":       vp.stage,
		"facts":       facts,
		"ballots":     ballots,
		"votes":       votes,
		"finished_at": vp.finishedAt,
		"is_closed":   vp.closed,
	}

	if vp.majority != nil {
		m["majority"] = vp.majority
	}

	return bson.Marshal(encoder.MergeBSONM(
		encoder.NewBSONHintedDoc(vp.Hint()),
		m,
	))
}

type VoteproofV0UnpackBSON struct { // nolint
	HT Height         `bson:"height"`
	RD Round          `bson:"round"`
	TH Threshold      `bson:"threshold"`
	RS VoteResultType `bson:"result"`
	ST Stage          `bson:"stage"`
	MJ bson.Raw       `bson:"majority"`
	FS [][2]bson.Raw  `bson:"facts"`
	BS [][2]bson.Raw  `bson:"ballots"`
	VS [][2]bson.Raw  `bson:"votes"`
	FA time.Time      `bson:"finished_at"`
	CL bool           `bson:"is_closed"`
}

func (vp *VoteproofV0) UnpackBSON(b []byte, enc *encoder.BSONEncoder) error { // nolint
	var vpp VoteproofV0UnpackBSON
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
		vpp.FA,
		vpp.CL,
	)
}

type VoteproofNodeFactPackBSON struct {
	FC valuehash.Hash `bson:"fact"`
	FS key.Signature  `bson:"fact_signature"`
	SG key.Publickey  `bson:"signer"`
}

func (vf VoteproofNodeFact) MarshalBSON() ([]byte, error) {
	return bson.Marshal(VoteproofNodeFactPackBSON{
		FC: vf.fact,
		FS: vf.factSignature,
		SG: vf.signer,
	})
}

type VoteproofNodeFactUnpackBSON struct {
	FC bson.Raw      `bson:"fact"`
	FS key.Signature `bson:"fact_signature"`
	SG bson.Raw      `bson:"signer"`
}

func (vf *VoteproofNodeFact) UnpackBSON(b []byte, enc *encoder.BSONEncoder) error {
	var vpp VoteproofNodeFactUnpackBSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	return vf.unpack(enc, vpp.FC, vpp.FS, vpp.SG)
}
