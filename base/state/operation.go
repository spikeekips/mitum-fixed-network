package state

import "github.com/spikeekips/mitum/util/errors"

var IgnoreOperationProcessingError = errors.NewError("ignore operation processing")

type OperationProcesser interface {
	ProcessOperation(
		getState func(key string) (StateUpdater, bool, error),
		setState func(StateUpdater) error,
	) error
}
