package block

import (
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/tree"
)

func (bm *BlockV0) unpack(enc encoder.Encoder, bmf, bco, bot, bops, bstt, bsts []byte) error {
	if m, err := DecodeManifest(bmf, enc); err != nil {
		return err
	} else if mv, ok := m.(ManifestV0); !ok {
		return util.WrongTypeError.Errorf("not ManifestV0: type=%T", m)
	} else {
		bm.ManifestV0 = mv
	}

	if m, err := DecodeConsensusInfo(bco, enc); err != nil {
		return err
	} else if mv, ok := m.(ConsensusInfoV0); !ok {
		return util.WrongTypeError.Errorf("not ConsensusInfoV0: type=%T", m)
	} else {
		bm.ci = mv
	}

	var err error
	bm.operationsTree, err = tree.DecodeFixedTree(bot, enc)
	if err != nil {
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

	tr, err := tree.DecodeFixedTree(bstt, enc)
	if err != nil {
		return err
	}
	bm.statesTree = tr

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
