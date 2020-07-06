package operation

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

const MaxTokenSize = 100

type BaseOperationFact interface {
	Fact() base.Fact
	Token() []byte
}

type Operation interface {
	isvalid.IsValider
	hint.Hinter
	valuehash.Hasher
	valuehash.HashGenerator
	BaseOperationFact
	Signs() []FactSign
}

func IsValidOperation(op Operation, networkID []byte) error {
	if err := op.Hint().IsValid(nil); err != nil {
		return err
	}

	if l := len(op.Token()); l < 1 {
		return isvalid.InvalidError.Errorf("Operation has empty token")
	} else if l > MaxTokenSize {
		return isvalid.InvalidError.Errorf("Operation token size too large: %d > %d", l, MaxTokenSize)
	}

	if err := op.Fact().IsValid(networkID); err != nil {
		return err
	}

	if len(op.Signs()) < 1 {
		return isvalid.InvalidError.Errorf("empty Signs()")
	}

	for i := range op.Signs() {
		fs := op.Signs()[i]
		if err := fs.IsValid(networkID); err != nil {
			return err
		} else if err := IsValidFactSign(op.Fact(), fs, networkID); err != nil {
			return err
		}
	}

	if h, err := op.GenerateHash(); err != nil {
		return err
	} else if !h.Equal(op.Hash()) {
		return isvalid.InvalidError.Errorf("wrong Opeartion hash")
	}

	return nil
}
