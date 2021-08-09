package base

import (
	"time"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
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
	HT Height           `bson:"height"`
	RD Round            `bson:"round"`
	SS []AddressDecoder `bson:"suffrages"`
	TH ThresholdRatio   `bson:"threshold"`
	RS VoteResultType   `bson:"result"`
	ST Stage            `bson:"stage"`
	MJ bson.Raw         `bson:"majority"`
	FS bson.Raw         `bson:"facts"`
	VS bson.Raw         `bson:"votes"`
	FA time.Time        `bson:"finished_at"`
	CL bool             `bson:"is_closed"`
}

func (vp *VoteproofV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var vpp VoteproofV0UnpackBSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
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
		vpp.FS,
		vpp.VS,
		vpp.FA,
		vpp.CL,
	)
}
