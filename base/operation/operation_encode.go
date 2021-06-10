package operation

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (bo *BaseOperation) unpack(enc encoder.Encoder, ht hint.Hint, h valuehash.Hash, fc []byte, fs [][]byte) error {
	if hinter, err := base.DecodeFact(enc, fc); err != nil {
		return err
	} else if f, ok := hinter.(OperationFact); !ok {
		return xerrors.Errorf("not OperationFact, %T", hinter)
	} else {
		bo.fact = f
	}

	ufs := make([]FactSign, len(fs))
	for i := range fs {
		f, err := DecodeFactSign(enc, fs[i])
		if err != nil {
			return err
		}
		ufs[i] = f
	}

	bo.ht = ht
	bo.h = h
	bo.fs = ufs

	return nil
}
