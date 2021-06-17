package state

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (st *StateV0) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	key string,
	bValue []byte,
	previousHeight base.Height,
	height base.Height,
	ops []valuehash.Bytes,
) error {
	value, err := DecodeValue(bValue, enc)
	if err != nil {
		return err
	}

	uops := make([]valuehash.Hash, len(ops))
	for i := range ops {
		uops[i] = ops[i]
	}

	st.h = h
	st.key = key
	st.value = value
	st.previousHeight = previousHeight
	st.height = height
	st.operations = uops

	return nil
}
