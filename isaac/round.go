package isaac

import (
	"bytes"
	"encoding/binary"
)

// Round is used to vote by ballot.
type Round uint64

// Uint64 returns int64 of height.
func (rn Round) Uint64() uint64 {
	return uint64(rn)
}

func (rn Round) Bytes() []byte {
	b := new(bytes.Buffer)
	_ = binary.Write(b, binary.LittleEndian, uint64(rn))

	return b.Bytes()
}
