package isvalid

import "github.com/spikeekips/mitum/util/errors"

var InvalidError = errors.NewError("invalid")

type IsValider interface {
	IsValid([]byte) error
}
