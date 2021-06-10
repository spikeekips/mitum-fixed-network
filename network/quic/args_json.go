package quicnetwork

import (
	"encoding/json"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type HashesArgsUnpackerJSON struct {
	Hashes []json.RawMessage
}

func (ha *HashesArgs) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uh HashesArgsUnpackerJSON
	if err := enc.Unmarshal(b, &uh); err != nil {
		return err
	}

	hs := make([]valuehash.Hash, len(uh.Hashes))
	for i := range uh.Hashes {
		h, err := valuehash.Decode(enc, uh.Hashes[i])
		if err != nil {
			return err
		}
		hs[i] = h
	}

	ha.Hashes = hs

	return nil
}
