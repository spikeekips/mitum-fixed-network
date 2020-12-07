package state

import (
	"github.com/spikeekips/mitum/util/valuehash"
)

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
