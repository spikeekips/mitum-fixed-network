package ballot

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (pr ProposalV0) MarshalBSON() ([]byte, error) {
	m := PackBaseBallotV0BSON(pr)

	m["facts"] = pr.facts
	m["seals"] = pr.seals

	return bsonenc.Marshal(m)
}

type ProposalV0UnpackerBSON struct {
	FS []valuehash.Bytes `bson:"facts"`
	SL []valuehash.Bytes `bson:"seals"`
}

func (pr *ProposalV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	bb, bf, err := pr.BaseBallotV0.unpackBSON(b, enc)
	if err != nil {
		return err
	}

	var npb ProposalV0UnpackerBSON
	if err := enc.Unmarshal(b, &npb); err != nil {
		return err
	}

	fs := make([]valuehash.Hash, len(npb.FS))
	for i := range npb.FS {
		fs[i] = npb.FS[i]
	}

	seals := make([]valuehash.Hash, len(npb.SL))
	for i := range npb.SL {
		seals[i] = npb.SL[i]
	}

	return pr.unpack(enc, bb, bf, fs, seals)
}
