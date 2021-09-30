package operation

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	SealType   = hint.Type("seal")
	SealHint   = hint.NewHint(SealType, "v0.0.1")
	SealHinter = BaseSeal{BaseSeal: seal.NewBaseSealWithHint(SealHint)}
)

type Seal interface {
	seal.Seal
	Operations() []Operation
}

type SealUpdater interface {
	SetOperations([]Operation) SealUpdater
}

type BaseSeal struct {
	seal.BaseSeal
	ops []Operation
}

func NewBaseSeal(pk key.Privatekey, ops []Operation, networkID []byte) (BaseSeal, error) {
	if len(ops) < 1 {
		return BaseSeal{}, errors.Errorf("seal can not be generated without Operations")
	}

	sl := BaseSeal{
		BaseSeal: seal.NewBaseSealWithHint(SealHint),
		ops:      ops,
	}

	sl.GenerateBodyHashFunc = func() (valuehash.Hash, error) {
		return valuehash.NewSHA256(sl.BodyBytes()), nil
	}

	if err := sl.Sign(pk, networkID); err != nil {
		return BaseSeal{}, err
	}

	return sl, nil
}

func (sl BaseSeal) IsValid(networkID []byte) error {
	if l := len(sl.ops); l < 1 {
		return isvalid.InvalidError.Errorf("empty operations")
	}

	if err := sl.BaseSeal.IsValid(networkID); err != nil {
		return err
	}

	for _, op := range sl.ops {
		if err := op.IsValid(networkID); err != nil {
			return err
		}
	}

	return nil
}

func (sl BaseSeal) BodyBytes() []byte {
	bl := make([][]byte, len(sl.ops)+1)
	bl[0] = sl.BaseSeal.BodyBytes()

	for i := range sl.ops {
		bl[i+1] = sl.ops[i].Hash().Bytes()
	}

	return util.ConcatBytesSlice(bl...)
}

func (sl BaseSeal) Operations() []Operation {
	return sl.ops
}

func (sl BaseSeal) SetOperations(ops []Operation) SealUpdater {
	sl.ops = ops

	return sl
}
