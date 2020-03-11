package valuehash

import (
	"fmt"

	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
)

var EmptyHashError = errors.NewError("empty hash")

type Hash interface {
	// NOTE usually String() value is the base58 encoded of Bytes()
	fmt.Stringer
	hint.Hinter
	isvalid.IsValider
	Size() int
	Bytes() []byte
	Equal(Hash) bool
}
