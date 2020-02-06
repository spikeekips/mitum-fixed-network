package isaac

import "github.com/spikeekips/mitum/errors"

var (
	InvalidError = errors.NewError("invalid found")
)

// IsValider is interface for checking validity.
type IsValider interface {
	IsValid() error
}
