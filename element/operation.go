package element

import (
	"github.com/spikeekips/mitum/common"
)

type OperationType interface {
	common.Marshaler
}

type OperationValue interface {
	common.Marshaler
}

type OperationOptions interface {
	common.Marshaler

	Get(string) interface{}
	Set(string) interface{}
}

type Operation interface {
	common.Marshaler

	Type() OperationType
	Value() OperationValue
	Options() OperationOptions
	Target() common.Address
}
