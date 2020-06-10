package quicnetwork

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base/valuehash"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
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
		if h, err := valuehash.Decode(enc, uh.Hashes[i]); err != nil {
			return err
		} else {
			hs[i] = h
		}
	}

	ha.Hashes = hs

	return nil
}
