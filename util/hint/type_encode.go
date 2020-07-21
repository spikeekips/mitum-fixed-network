package hint

import (
	"encoding/hex"
)

func (ty *Type) unpack(s string) error {
	var t [2]byte
	if d, err := hex.DecodeString(s); err != nil {
		return err
	} else {
		copy(t[:], d)
	}

	nt := Type(t)

	*ty = nt

	return nil
}
