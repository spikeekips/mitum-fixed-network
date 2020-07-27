package policy

import (
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (spo *SetPolicyV0) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	bfs [][]byte,
	token []byte,
	bPolicy []byte,
) error {
	var po PolicyV0
	if p, err := DecodePolicyV0(enc, bPolicy); err != nil {
		return err
	} else {
		po = p
	}

	fs := make([]operation.FactSign, len(bfs))
	for i := range bfs {
		if f, err := operation.DecodeFactSign(enc, bfs[i]); err != nil {
			return err
		} else {
			fs[i] = f
		}
	}

	spo.h = h
	spo.fs = fs
	spo.SetPolicyFactV0 = SetPolicyFactV0{
		PolicyV0: po,
		token:    token,
	}

	return nil
}
