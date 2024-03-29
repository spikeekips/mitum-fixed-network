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

func encodeWithEncoder(b []byte, enc encoder.Encoder) []byte {
	h := make([]byte, hint.MaxHintLength)
	copy(h, enc.Hint().Bytes())

	return util.ConcatBytesSlice(h, b)
}

func marshal(i interface{}, enc encoder.Encoder) ([]byte, error) {
	b, err := enc.Marshal(i)
	if err != nil {
		return nil, err
	}

	return encodeWithEncoder(b, enc), nil
}

func mergeError(err error) error {
	if err == nil {
		return nil
	}

	if err == leveldbErrors.ErrNotFound {
		return util.NotFoundError.Merge(err)
	}

	return storage.MergeStorageError(err)
}
