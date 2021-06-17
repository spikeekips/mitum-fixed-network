package quicnetwork

import (
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (ha *HashesArgs) UnmarshalBSON(b []byte) error {
	var uh []valuehash.Bytes
	if err := bson.Unmarshal(b, &uh); err != nil {
		return err
	}

	ha.Hashes = make([]valuehash.Hash, len(uh))
	for i := range uh {
		ha.Hashes[i] = uh[i]
	}

	return nil
}
