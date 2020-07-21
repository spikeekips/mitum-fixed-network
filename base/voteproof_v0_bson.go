package base

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

type VoteproofV0FactBSONPacker struct {
	H valuehash.Hash
	F Fact
}

type VoteproofV0FactBSONUnpacker struct {
	H valuehash.Bytes
	F bson.Raw
}

func (vv VoteproofV0FactBSONUnpacker) Hash() valuehash.Bytes {
	return vv.H
}

func (vv VoteproofV0FactBSONUnpacker) Fact() []byte {
	return vv.F
}

type VoteproofV0BallotBSONPacker struct {
	H valuehash.Hash
	A Address
}

type VoteproofV0BallotBSONUnpacker struct {
	H valuehash.Bytes
	A bson.Raw
}

func (vv VoteproofV0BallotBSONUnpacker) Hash() valuehash.Bytes {
	return vv.H
}

func (vv VoteproofV0BallotBSONUnpacker) Address() []byte {
	return vv.A
}

func (vp VoteproofV0) MarshalBSON() ([]byte, error) {
	m := bson.M{
		"height":      vp.height,
		"round":       vp.round,
		"suffrages":   vp.suffrages,
		"threshold":   vp.thresholdRatio,
		"result":      vp.result,
		"stage":       vp.stage,
		"facts":       vp.facts,
		"votes":       vp.votes,
		"finished_at": vp.finishedAt,
		"is_closed":   vp.closed,
	}

	if vp.majority != nil {
		m["majority"] = vp.majority
	}

	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(vp.Hint()),
		m,
	))
}

type VoteproofV0UnpackBSON struct { // nolint
	HT Height         `bson:"height"`
	RD Round          `bson:"round"`
	SS []bson.Raw     `bson:"suffrages"`
	TH ThresholdRatio `bson:"threshold"`
	RS VoteResultType `bson:"result"`
	ST Stage          `bson:"stage"`
	MJ bson.Raw       `bson:"majority"`
	FS []bson.Raw     `bson:"facts"`
	VS []bson.Raw     `bson:"votes"`
	FA time.Time      `bson:"finished_at"`
	CL bool           `bson:"is_closed"`
}

func (vp *VoteproofV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var vpp VoteproofV0UnpackBSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	ss := make([][]byte, len(vpp.SS))
	for i := range vpp.SS {
		ss[i] = vpp.SS[i]
	}

	fs := make([][]byte, len(vpp.FS))
	for i := range vpp.FS {
		fs[i] = vpp.FS[i]
	}

	vs := make([][]byte, len(vpp.VS))
	for i := range vpp.VS {
		vs[i] = vpp.VS[i]
	}

	return vp.unpack(
		enc,
		vpp.HT,
		vpp.RD,
		ss,
		vpp.TH,
		vpp.RS,
		vpp.ST,
		vpp.MJ,
		fs,
		vs,
		vpp.FA,
		vpp.CL,
	)
}

type VoteproofNodeFactPackBSON struct {
	AD Address        `bson:"address"`
	BT valuehash.Hash `bson:"ballot"`
	FC valuehash.Hash `bson:"fact"`
	FS key.Signature  `bson:"fact_signature"`
	SG key.Publickey  `bson:"signer"`
}

func (vf VoteproofNodeFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(VoteproofNodeFactPackBSON{
		AD: vf.address,
		BT: vf.ballot,
		FC: vf.fact,
		FS: vf.factSignature,
		SG: vf.signer,
	})
}

type VoteproofNodeFactUnpackBSON struct {
	AD bson.Raw             `bson:"address"`
	BT valuehash.Bytes      `bson:"ballot"`
	FC valuehash.Bytes      `bson:"fact"`
	FS key.Signature        `bson:"fact_signature"`
	SG encoder.HintedString `bson:"signer"`
}

func (vf *VoteproofNodeFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var vpp VoteproofNodeFactUnpackBSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	return vf.unpack(enc, vpp.AD, vpp.BT, vpp.FC, vpp.FS, vpp.SG)
}
