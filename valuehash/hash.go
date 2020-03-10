package valuehash

import (
	"fmt"

	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
)

var EmptyHashError = errors.NewError("empty hash")

type Hash interface {
	fmt.Stringer // TODO remove 'string' or 'hash' from json
	hint.Hinter
	isvalid.IsValider
	Size() int
	Bytes() []byte
	Equal(Hash) bool
}
