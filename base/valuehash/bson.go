package valuehash

import (
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
)

type BSONHash struct {
	HI   hint.Hint `bson:"_hint"`
	Hash string    `bson:"hash"`
}

func marshalBSON(h Hash) ([]byte, error) {
	return bsonencoder.Marshal(BSONHash{
		HI:   h.Hint(),
		Hash: h.String(),
	})
}

func unmarshalBSON(b []byte) (BSONHash, error) {
	var bh BSONHash
	if err := bsonencoder.Unmarshal(b, &bh); err != nil {
		return BSONHash{}, err
	}

	return bh, nil
}
