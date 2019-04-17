package element

import "github.com/spikeekips/mitum/common"

type OperationType interface {
	String() string
	Bytes() []byte
}

type OperationValue interface {
	Bytes() []byte
}

type Operation interface {
	Target() common.Address
	Type() OperationType
	Value() OperationValue
}
