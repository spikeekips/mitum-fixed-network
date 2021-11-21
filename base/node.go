package base

import (
	"fmt"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

type Node interface {
	fmt.Stringer
	isvalid.IsValider
	hint.Hinter
	Address() Address
	Publickey() key.Publickey
}
