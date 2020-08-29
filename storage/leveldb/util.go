package leveldbstorage

import (
	leveldbErrors "github.com/syndtr/goleveldb/leveldb/errors"

	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func loadHint(b []byte) (hint.Hint, []byte, error) {
	var ht hint.Hint
	if h, err := hint.NewHintFromBytes(b[:hint.MaxHintSize]); err != nil {
		return hint.Hint{}, nil, err
	} else {
		ht = h
	}

	return ht, b[hint.MaxHintSize:], nil
}

func encodeWithEncoder(enc encoder.Encoder, b []byte) []byte {
	h := make([]byte, hint.MaxHintSize)
	copy(h, enc.Hint().Bytes())

	return util.ConcatBytesSlice(h, b)
}

func marshal(enc encoder.Encoder, i interface{}) ([]byte, error) {
	b, err := enc.Marshal(i)
	if err != nil {
		return nil, err
	}

	return encodeWithEncoder(enc, b), nil
}

func wrapError(err error) error {
	if err == nil {
		return nil
	}

	if err == leveldbErrors.ErrNotFound {
		return storage.NotFoundError.Wrap(err)
	}

	return storage.WrapStorageError(err)
}
