package key

import (
	"fmt"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

const KeyTypeSize = 3

type Key interface {
	fmt.Stringer
	hint.Hinter
	util.Byter
	isvalid.IsValider
	Equal(Key) bool
}

type Privatekey interface {
	Key
	Publickey() Publickey
	Sign([]byte) (Signature, error)
}

type Publickey interface {
	Key
	Verify([]byte, Signature) error
}
