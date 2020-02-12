package isvalid

import "github.com/spikeekips/mitum/errors"

var (
	InvalidError = errors.NewError("invalid found")
)

type IsValider interface {
	IsValid([]byte) error
}
