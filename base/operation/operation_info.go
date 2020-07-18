package operation

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	OperationInfoV0Type = hint.MustNewType(0x01, 0x61, "operation-info-v0")
	OperationInfoV0Hint = hint.MustHint(OperationInfoV0Type, "0.0.1")
)

type OperationInfo interface {
	isvalid.IsValider
	hint.Hinter
	util.Byter
	Operation() valuehash.Hash
	Seal() valuehash.Hash
}

type OperationInfoV0 struct {
	oh valuehash.Hash
	sh valuehash.Hash
	op Operation
}

func NewOperationInfoV0(op Operation, sh valuehash.Hash) OperationInfoV0 {
	return OperationInfoV0{
		oh: op.Hash(),
		sh: sh,
		op: op,
	}
}

func (oi OperationInfoV0) Hint() hint.Hint {
	return OperationInfoV0Hint
}

func (oi OperationInfoV0) IsValid([]byte) error {
	if err := oi.oh.IsValid(nil); err != nil {
		return err
	}

	return oi.sh.IsValid(nil)
}

func (oi OperationInfoV0) Operation() valuehash.Hash {
	return oi.oh
}

func (oi OperationInfoV0) RawOperation() Operation {
	return oi.op
}

func (oi OperationInfoV0) Seal() valuehash.Hash {
	return oi.sh
}

func (oi OperationInfoV0) Bytes() []byte {
	return util.ConcatBytesSlice(oi.oh.Bytes(), oi.sh.Bytes())
}
