package quicnetwork

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

type HashesArgsUnpackerJSON struct {
	Hashes []valuehash.Bytes
}

func (ha *HashesArgs) UnmarshalJSON(b []byte) error {
	var uh HashesArgsUnpackerJSON
	if err := jsonenc.Unmarshal(b, &uh); err != nil {
		return err
	}

	hs := make([]valuehash.Hash, len(uh.Hashes))
	for i := range uh.Hashes {
		hs[i] = uh.Hashes[i]
	}

	ha.Hashes = hs

	return nil
}
