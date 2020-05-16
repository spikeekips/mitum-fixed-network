package hint

import (
	"encoding/hex"
)

func (ty *Type) unpack(c string) error {
	var t [2]byte
	if d, err := hex.DecodeString(c); err != nil {
		return err
	} else {
		copy(t[:], d)
	}

	nt := Type(t)

	*ty = nt

	return nil
}
