package quicnetwork

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (ha *HashesArgs) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uh []valuehash.Bytes
	if err := enc.Unmarshal(b, &uh); err != nil {
		return err
	}

	ha.Hashes = make([]valuehash.Hash, len(uh))
	for i := range uh {
		ha.Hashes[i] = uh[i]
	}

	return nil
}
