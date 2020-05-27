package util

import "github.com/spikeekips/mitum/util/errors"

var CheckerNilError = errors.NewError("")

type CheckerFunc func() (bool, error)

type Checker struct {
	name string
	fns  []CheckerFunc
}

func NewChecker(name string, fns []CheckerFunc) Checker {
	return Checker{
		name: name,
		fns:  fns,
	}
}

func (ck Checker) Check() error {
	for _, fn := range ck.fns {
		keep, err := fn()
		if err != nil {
			return err
		} else if !keep {
			return nil
		}
	}

	return nil
}
