package valuehash

import (
	"github.com/spikeekips/mitum/util/encoder"
)

func Decode(enc encoder.Encoder, b []byte) (Hash, error) {
	var bt Bytes
	if err := enc.Unmarshal(b, &bt); err != nil {
		return nil, err
	}

	return bt, nil
}
