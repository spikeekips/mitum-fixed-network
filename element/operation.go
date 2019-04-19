package element

import (
	"github.com/spikeekips/mitum/common"
)

type OperationType interface {
	common.BinaryEncoder
	common.TextEncoder
}

type OperationValue interface {
	common.BinaryEncoder
	common.TextEncoder
}

type OperationOptions interface {
	common.BinaryEncoder
	common.TextEncoder

	Get(string) interface{}
	Set(string) interface{}
}

type Operation interface {
	common.BinaryEncoder
	common.TextEncoder

	Type() OperationType
	Value() OperationValue
	Options() OperationOptions
	Target() common.Address
}
