package operation

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
)

const MaxTokenSize = 100

type BaseOperationFact interface {
	EmbededFact
	Signer() key.Publickey
	Token() []byte
}

type Operation interface {
	isvalid.IsValider
	hint.Hinter
	valuehash.Hasher
	valuehash.HashGenerator
	BaseOperationFact
}

func IsValidOperation(op Operation, b []byte) error {
	if err := op.Hint().IsValid(nil); err != nil {
		return err
	}

	if l := len(op.Token()); l < 1 {
		return isvalid.InvalidError.Errorf("Operation has empty token")
	} else if l > MaxTokenSize {
		return isvalid.InvalidError.Errorf("Operation token size too large: %d > %d", l, MaxTokenSize)
	}

	if err := op.Fact().IsValid(b); err != nil {
		return err
	}

	if err := IsValidEmbededFact(op.Signer(), op, b); err != nil {
		return err
	}

	if h, err := op.GenerateHash(); err != nil {
		return err
	} else if !h.Equal(op.Hash()) {
		return isvalid.InvalidError.Errorf("wrong Opeartion hash")
	}

	return nil
}
