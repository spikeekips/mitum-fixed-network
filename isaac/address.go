package isaac

import (
	"fmt"

	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
)

var (
	EmptyAddressError = errors.NewError("empty address")
)

// Address represents the address of account.
type Address interface {
	fmt.Stringer
	isvalid.IsValider
	hint.Hinter
	Equal(Address) bool
	Bytes() []byte
}
