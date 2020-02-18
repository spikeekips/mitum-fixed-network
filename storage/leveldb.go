package storage

import (
	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/util"
)

func LeveldbLoadHint(b []byte) (hint.Hint, []byte, error) {
	var ht hint.Hint
	if h, err := hint.NewHintFromBytes(b[:hint.MaxHintSize]); err != nil {
		return hint.Hint{}, nil, err
	} else {
		ht = h
	}

	return ht, b[hint.MaxHintSize:], nil
}

func LeveldbDataWithEncoder(enc encoder.Encoder, b []byte) []byte {
	h := make([]byte, hint.MaxHintSize)
	copy(h, enc.Hint().Bytes())

	return util.ConcatSlice([][]byte{h, b})
}
