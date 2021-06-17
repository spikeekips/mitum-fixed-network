package base

import (
	"github.com/spikeekips/mitum/base/key"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (vf BaseVoteproofNodeFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(bsonenc.NewHintedDoc(vf.Hint()), bson.M{
		"address":        vf.address,
		"ballot":         vf.ballot,
		"fact":           vf.fact,
		"fact_signature": vf.factSignature,
		"signer":         vf.signer,
	}))
}

type BaseVoteproofNodeFactUnpackBSON struct {
	AD AddressDecoder       `bson:"address"`
	BT valuehash.Bytes      `bson:"ballot"`
	FC valuehash.Bytes      `bson:"fact"`
	FS key.Signature        `bson:"fact_signature"`
	SG key.PublickeyDecoder `bson:"signer"`
}

func (vf *BaseVoteproofNodeFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var vpp BaseVoteproofNodeFactUnpackBSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	return vf.unpack(enc, vpp.AD, vpp.BT, vpp.FC, vpp.FS, vpp.SG)
}
