package isaac

import (
	"fmt"

	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/util"
)

var EmptyAddressError = errors.NewError("empty address")

// Address represents the address of account.
type Address interface {
	fmt.Stringer
	isvalid.IsValider
	hint.Hinter
	util.Byter
	Equal(Address) bool
}
