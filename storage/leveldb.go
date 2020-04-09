package storage

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	leveldbErrors "github.com/syndtr/goleveldb/leveldb/errors"
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

	return util.ConcatBytesSlice(h, b)
}

func LeveldbMarshal(enc encoder.Encoder, i interface{}) ([]byte, error) {
	b, err := enc.Encode(i)
	if err != nil {
		return nil, err
	}

	return LeveldbDataWithEncoder(enc, b), nil
}

func LeveldbWrapError(err error) error {
	if err == nil {
		return nil
	}

	if err == leveldbErrors.ErrNotFound {
		return NotFoundError.Wrap(err)
	}

	return WrapError(err)
}
