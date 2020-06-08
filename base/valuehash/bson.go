package valuehash

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
)

type BSONHash struct {
	HI   hint.Hint `bson:"_hint"`
	Hash string    `bson:"hash"`
}

func marshalBSON(h Hash) ([]byte, error) {
	return bsonenc.Marshal(BSONHash{
		HI:   h.Hint(),
		Hash: h.String(),
	})
}

func unmarshalBSON(b []byte) (BSONHash, error) {
	var bh BSONHash
	if err := bsonenc.Unmarshal(b, &bh); err != nil {
		return BSONHash{}, err
	}

	return bh, nil
}
