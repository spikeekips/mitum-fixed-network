package base

import (
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Fact interface {
	isvalid.IsValider
	hint.Hinter
	valuehash.Hasher
}
