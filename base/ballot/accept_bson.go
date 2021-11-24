package ballot // nolint:dupl

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (fact ACCEPTFact) MarshalBSON() ([]byte, error) {
	return bson.Marshal(bsonenc.MergeBSONM(
		fact.BaseFact.packerBSON(),
		bson.M{
			"proposal":  fact.proposal,
			"new_block": fact.newBlock,
		}))
}

type ACCEPTFactUnpackerBSON struct {
	P valuehash.Bytes `bson:"proposal"`
	N valuehash.Bytes `bson:"new_block"`
}

func (fact *ACCEPTFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	if err := fact.BaseFact.unpackBSON(b, enc); err != nil {
		return err
	}

	var uf ACCEPTFactUnpackerBSON
	if err := enc.Unmarshal(b, &uf); err != nil {
		return err
	}

	fact.proposal = uf.P
	fact.newBlock = uf.N

	return nil
}
