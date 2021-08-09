package operation

import (
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

const MaxTokenSize = 100

type OperationFact interface {
	base.Fact
	Token() []byte
}

type Operation interface {
	isvalid.IsValider
	hint.Hinter
	valuehash.Hasher
	valuehash.HashGenerator
	Fact() base.Fact
	Signs() []FactSign
	LastSignedAt() time.Time
}

func IsValidOperation(op Operation, networkID []byte) error {
	if err := op.Hint().IsValid(nil); err != nil {
		return err
	}

	fact, ok := op.Fact().(OperationFact)
	if !ok {
		return isvalid.InvalidError.Errorf("wrong fact, %T found", op.Fact())
	}

	if l := len(fact.Token()); l < 1 {
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

	if !op.Hash().Equal(op.GenerateHash()) {
		return isvalid.InvalidError.Errorf("wrong Opeartion hash")
	}

	return nil
}

type BaseOperation struct {
	ht   hint.Hint
	fact OperationFact
	h    valuehash.Hash
	fs   []FactSign
}

func NewBaseOperation(ht hint.Hint, fact OperationFact, h valuehash.Hash, fs []FactSign) BaseOperation {
	return BaseOperation{
		ht:   ht,
		fact: fact,
		h:    h,
		fs:   fs,
	}
}

func NewBaseOperationFromFact(ht hint.Hint, fact OperationFact, fs []FactSign) (BaseOperation, error) {
	bo := BaseOperation{
		ht:   ht,
		fact: fact,
		fs:   fs,
	}

	bo.h = bo.GenerateHash()

	return bo, nil
}

func (bo BaseOperation) SetHash(h valuehash.Hash) BaseOperation {
	bo.h = h

	return bo
}

func (bo BaseOperation) SetHint(ht hint.Hint) BaseOperation {
	bo.ht = ht

	return bo
}

func (bo BaseOperation) Fact() base.Fact {
	return bo.fact
}

func (bo BaseOperation) Token() []byte {
	return bo.fact.Token()
}

func (bo BaseOperation) Hint() hint.Hint {
	return bo.ht
}

func (bo BaseOperation) Hash() valuehash.Hash {
	return bo.h
}

func (bo BaseOperation) GenerateHash() valuehash.Hash {
	bs := make([][]byte, len(bo.fs))
	for i := range bo.fs {
		bs[i] = bo.fs[i].Bytes()
	}

	e := util.ConcatBytesSlice(bo.Fact().Hash().Bytes(), util.ConcatBytesSlice(bs...))

	return valuehash.NewSHA256(e)
}

func (bo BaseOperation) Signs() []FactSign {
	return bo.fs
}

func (bo BaseOperation) IsValid(networkID []byte) error {
	return IsValidOperation(bo, networkID)
}

func (bo BaseOperation) AddFactSigns(fs ...FactSign) (FactSignUpdater, error) {
	for i := range bo.fs {
		bofs := bo.fs[i]

		var found bool
		for j := range fs {
			if bofs.Signer().Equal(fs[j].Signer()) {
				found = true
				break
			}
		}

		if found {
			return nil, errors.Errorf("already signed")
		}
	}

	bo.fs = append(bo.fs, fs...)
	bo.h = bo.GenerateHash()

	return bo, nil
}

func (bo BaseOperation) LastSignedAt() time.Time {
	return LastSignedAt(bo.fs)
}

func LastSignedAt(fs []FactSign) time.Time {
	if n := len(fs); n < 1 {
		return time.Time{}
	} else if n == 1 {
		return fs[0].SignedAt()
	}

	sort.Slice(fs, func(i, j int) bool {
		return fs[i].SignedAt().After(fs[j].SignedAt())
	})

	return fs[0].SignedAt()
}
