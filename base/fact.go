package base

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

type FactMode uint8

const (
	FNil      FactMode = 0x01
	FInStates FactMode = 0x02
)

type Fact interface {
	isvalid.IsValider
	hint.Hinter
	util.Byter
	valuehash.Hasher
}

func FactMode2bytes(m FactMode) []byte {
	return util.Uint8ToBytes(uint8(m))
}

func BytesToFactMode(b []byte) (FactMode, error) {
	if len(b) < 1 {
		return FNil, nil
	}

	if m, err := util.BytesToUint8(b); err != nil {
		return FNil, err
	} else {
		return FactMode(m), nil
	}
}
