package hint

import (
	"encoding/hex"
)

func (ty *Type) unpack(c, n string) error {
	var t [2]byte
	if d, err := hex.DecodeString(c); err != nil {
		return err
	} else {
		copy(t[:], d)
	}

	nt := Type(t)

	if t, err := typeByName(n); err != nil {
		return err
	} else if !nt.Equal(t) {
		return NewTypeDoesNotMatchError(t, nt)
	}

	*ty = nt

	return nil
}
