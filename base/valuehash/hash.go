package valuehash

import (
	"fmt"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/logging"
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
