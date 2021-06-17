package valuehash

import (
	"github.com/spikeekips/mitum/util/encoder"
)

func Decode(b []byte, enc encoder.Encoder) (Hash, error) {
	var bt Bytes
	if err := enc.Unmarshal(b, &bt); err != nil {
		return nil, err
	}

	return bt, nil
}
