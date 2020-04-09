package state

import (
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

type Value interface {
	util.Byter
	hint.Hinter
	valuehash.Hasher
	isvalid.IsValider
	Equal(Value) bool
	Interface() interface{}
	Set(interface{}) (Value, error)
}
