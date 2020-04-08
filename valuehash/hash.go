package valuehash

import (
	"fmt"

	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/util"
)

var EmptyHashError = errors.NewError("empty hash")

type Hash interface {
	// NOTE usually String() value is the base58 encoded of Bytes()
	fmt.Stringer
	hint.Hinter
	isvalid.IsValider
	util.Byter
	Size() int
	Equal(Hash) bool
	logging.LogHintedMarshaler
}

type Hasher interface {
	Hash() Hash
}

type HashGenerator interface {
	GenerateHash() (Hash, error)
}
