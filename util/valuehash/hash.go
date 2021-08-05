package valuehash

import (
	"fmt"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	EmptyHashError   = errors.NewError("empty hash")
	InvalidHashError = errors.NewError("invalid hash")
)

type Hash interface {
	isvalid.IsValider
	util.Byter
	// NOTE usually String() value is the base58 encoded of Bytes()
	fmt.Stringer
	Size() int
	Equal(Hash) bool
	Empty() bool
}

type Hasher interface {
	Hash() Hash
}

type HashGenerator interface {
	GenerateHash() Hash
}
