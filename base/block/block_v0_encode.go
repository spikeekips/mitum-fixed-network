package block

import (
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

func (bm *BlockV0) unpack(enc encoder.Encoder, bmf, bco, bot, bops, bstt, bsts []byte) error {
	if err := encoder.Decode(bmf, enc, &bm.ManifestV0); err != nil {
		return err
	}

	if err := encoder.Decode(bco, enc, &bm.ci); err != nil {
		return err
	}

	if err := encoder.Decode(bot, enc, &bm.operationsTree); err != nil {
		return err
	}

	hops, err := enc.DecodeSlice(bops)
	if err != nil {
		return err
	}

	ops := make([]operation.Operation, len(hops))
	for i := range hops {
		j, ok := hops[i].(operation.Operation)
		if !ok {
			return util.WrongTypeError.Errorf("expected operation.Operation, not %T", hops[i])
		}
		ops[i] = j
	}
	bm.operations = ops

	if err = encoder.Decode(bstt, enc, &bm.statesTree); err != nil {
		return err
	}

	hsts, err := enc.DecodeSlice(bsts)
	if err != nil {
		return err
	}

	sts := make([]state.State, len(hsts))
	for i := range hsts {
		j, ok := hsts[i].(state.State)
		if !ok {
			return util.WrongTypeError.Errorf("expected state.State, not %T", hops[i])
		}
		sts[i] = j
	}
	bm.states = sts

	return nil
}
