package isvalid

import "github.com/spikeekips/mitum/errors"

var InvalidError = errors.NewError("invalid")

type IsValider interface {
	IsValid([]byte) error
}
