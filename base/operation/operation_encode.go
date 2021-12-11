package operation

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (bo *BaseOperation) unpack(enc encoder.Encoder, h valuehash.Hash, fc []byte, fs []byte) error {
	if err := encoder.Decode(fc, enc, &bo.fact); err != nil {
		return err
	}

	hfs, err := enc.DecodeSlice(fs)
	if err != nil {
		return err
	}

	ufs := make([]base.FactSign, len(hfs))
	for i := range hfs {
		j, ok := hfs[i].(base.FactSign)
		if !ok {
			return util.WrongTypeError.Errorf("expected FactSign, not %T", hfs[i])
		}

		ufs[i] = j
	}

	bo.h = h
	bo.fs = ufs

	return nil
}
