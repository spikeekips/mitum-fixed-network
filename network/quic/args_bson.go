package quicnetwork

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/valuehash"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

type HashesArgsUnpackerBSON struct {
	Hashes []bson.Raw
}

func (ha *HashesArgs) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uh HashesArgsUnpackerBSON
	if err := enc.Unmarshal(b, &uh); err != nil {
		return err
	}

	hs := make([]valuehash.Hash, len(uh.Hashes))
	for i := range uh.Hashes {
		if h, err := valuehash.Decode(enc, uh.Hashes[i]); err != nil {
			return err
		} else {
			hs[i] = h
		}
	}

	ha.Hashes = hs

	return nil
}
