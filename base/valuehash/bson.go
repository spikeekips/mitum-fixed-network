package valuehash

import (
	"github.com/spikeekips/mitum/util/hint"
	"go.mongodb.org/mongo-driver/bson"
)

type BSONHash struct {
	HI   hint.Hint `bson:"_hint"`
	Hash string    `bson:"hash"`
}

func marshalBSON(h Hash) ([]byte, error) {
	return bson.Marshal(BSONHash{
		HI:   h.Hint(),
		Hash: h.String(),
	})
}

func unmarshalBSON(b []byte) (BSONHash, error) {
	var bh BSONHash
	if err := bson.Unmarshal(b, &bh); err != nil {
		return BSONHash{}, err
	}

	return bh, nil
}
