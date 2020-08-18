package state

import (
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/valuehash"
)

var IgnoreOperationProcessingError = errors.NewError("ignore operation processing")

type Processor interface {
	Process(
		getState func(key string) (StateUpdater, bool, error),
		setState func(valuehash.Hash, ...StateUpdater) error,
	) error
}

type PreProcessor interface {
	PreProcess(
		getState func(key string) (StateUpdater, bool, error),
		setState func(valuehash.Hash, ...StateUpdater) error,
	) (Processor, error)
}
