package operation

import (
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func (sl *BaseSeal) unpack(enc encoder.Encoder, ub seal.BaseSeal, bops []byte) error {
	sl.BaseSeal = ub

	hops, err := enc.DecodeSlice(bops)
	if err != nil {
		return err
	}

	sl.ops = make([]Operation, len(hops))
	for i := range hops {
		j, ok := hops[i].(Operation)
		if !ok {
			return util.WrongTypeError.Errorf("expected Operation, not %T", hops[i])
		}

		sl.ops[i] = j
	}

	return nil
}
