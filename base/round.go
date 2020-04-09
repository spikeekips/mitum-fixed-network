package base

import (
	"github.com/spikeekips/mitum/util"
)

// Round is used to vote by ballot.
type Round uint64

// Uint64 returns int64 of height.
func (rn Round) Uint64() uint64 {
	return uint64(rn)
}

func (rn Round) Bytes() []byte {
	return util.Uint64ToBytes(uint64(rn))
}
