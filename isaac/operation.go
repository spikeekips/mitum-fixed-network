package isaac

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
	"golang.org/x/xerrors"
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
	BaseOperationFact
	Hash() valuehash.Hash
	GenerateHash([]byte) (valuehash.Hash, error)
}

func IsValidOperation(op Operation, b []byte) error {
	if err := op.Hint().IsValid(b); err != nil {
		return err
	}

	if l := len(op.Token()); l < 1 {
		return xerrors.Errorf("Operation has empty token")
	} else if l > MaxTokenSize {
		return xerrors.Errorf("Operation token size too large: %d > %d", l, MaxTokenSize)
	}

	if err := op.Fact().IsValid(b); err != nil {
		return err
	}

	if err := IsValidEmbededFact(op.Signer(), op, b); err != nil {
		return err
	}

	if h, err := op.GenerateHash(b); err != nil {
		return err
	} else if !h.Equal(op.Hash()) {
		return xerrors.Errorf("wrong Opeartion hash")
	}

	return nil
}
