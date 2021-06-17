package operation

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (bo *BaseOperation) unpack(enc encoder.Encoder, ht hint.Hint, h valuehash.Hash, fc []byte, fs []byte) error {
	if hinter, err := base.DecodeFact(fc, enc); err != nil {
		return err
	} else if f, ok := hinter.(OperationFact); !ok {
		return xerrors.Errorf("not OperationFact, %T", hinter)
	} else {
		bo.fact = f
	}

	hfs, err := enc.DecodeSlice(fs)
	if err != nil {
		return err
	}

	ufs := make([]FactSign, len(hfs))
	for i := range hfs {
		j, ok := hfs[i].(FactSign)
		if !ok {
			return util.WrongTypeError.Errorf("expected FactSign, not %T", hfs[i])
		}

		ufs[i] = j
	}

	bo.ht = ht
	bo.h = h
	bo.fs = ufs

	return nil
}
