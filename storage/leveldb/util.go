package leveldbstorage

import (
	"bytes"

	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	leveldbErrors "github.com/syndtr/goleveldb/leveldb/errors"
)

func loadHint(b []byte) (hint.Hint, []byte, error) {
	var ht hint.Hint
	if h, err := hint.ParseHint(string(bytes.TrimRight(b[:hint.MaxHintLength], "\x00"))); err != nil {
		return hint.Hint{}, nil, err
	} else {
		ht = h
	}

	return ht, b[hint.MaxHintLength:], nil
}

func encodeWithEncoder(enc encoder.Encoder, b []byte) []byte {
	h := make([]byte, hint.MaxHintLength)
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
		return util.NotFoundError.Wrap(err)
	}

	return storage.WrapStorageError(err)
}
