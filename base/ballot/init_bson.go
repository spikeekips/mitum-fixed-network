package ballot

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (fact INITFact) MarshalBSON() ([]byte, error) {
	return bson.Marshal(bsonenc.MergeBSONM(
		fact.BaseFact.packerBSON(),
		bson.M{
			"previous_block": fact.previousBlock,
		}))
}

type INITFactUnpackerBSON struct {
	P valuehash.Bytes `bson:"previous_block"`
}

func (fact *INITFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	if err := fact.BaseFact.unpackBSON(b, enc); err != nil {
		return err
	}

	var uf INITFactUnpackerBSON
	if err := enc.Unmarshal(b, &uf); err != nil {
		return err
	}

	fact.previousBlock = uf.P

	return nil
}
