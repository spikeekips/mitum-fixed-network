package deploy

import (
	"time"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

func (dk DeployKey) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bson.M{
		"key":      dk.k,
		"added_at": dk.addedAt,
	})
}

type DeployKeyUnpackerBSON struct {
	K  string    `bson:"key"`
	AA time.Time `bson:"added_at"`
}

func (dk *DeployKey) UnmarshalBSON(b []byte) error {
	var udk DeployKeyUnpackerBSON
	if err := bsonenc.Unmarshal(b, &udk); err != nil {
		return err
	}

	return dk.unpack(udk.K, udk.AA)
}
