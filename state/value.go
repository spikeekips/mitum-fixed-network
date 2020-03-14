package state

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
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
