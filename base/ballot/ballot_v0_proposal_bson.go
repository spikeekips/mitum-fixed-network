package ballot

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
)

func (pr ProposalV0) MarshalBSON() ([]byte, error) {
	m := PackBaseBallotV0BSON(pr)

	m["operations"] = pr.operations
	m["seals"] = pr.seals

	return bsonencoder.Marshal(m)
}

type ProposalV0UnpackerBSON struct {
	OP []bson.Raw `bson:"operations"`
	SL []bson.Raw `bson:"seals"`
}

func (pr *ProposalV0) UnpackBSON(b []byte, enc *bsonencoder.Encoder) error {
	bb, bf, err := pr.BaseBallotV0.unpackBSON(b, enc)
	if err != nil {
		return err
	}

	var npb ProposalV0UnpackerBSON
	if err := enc.Unmarshal(b, &npb); err != nil {
		return err
	}

	ops := make([][]byte, len(npb.OP))
	for i, r := range npb.OP {
		ops[i] = r
	}

	seals := make([][]byte, len(npb.SL))
	for i, r := range npb.SL {
		seals[i] = r
	}

	return pr.unpack(enc, bb, bf, ops, seals)
}
