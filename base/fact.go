package base

import (
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

type Fact interface {
	isvalid.IsValider
	hint.Hinter
	util.Byter
	valuehash.Hasher
}
