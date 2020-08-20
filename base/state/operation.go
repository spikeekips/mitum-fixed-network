package state

import (
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/valuehash"
)

var IgnoreOperationProcessingError = errors.NewError("ignore operation processing")

type Processor interface {
	Process(
		getState func(key string) (State, bool, error),
		setState func(valuehash.Hash, ...State) error,
	) error
}

type PreProcessor interface {
	PreProcess(
		getState func(key string) (State, bool, error),
		setState func(valuehash.Hash, ...State) error,
	) (Processor, error)
}
