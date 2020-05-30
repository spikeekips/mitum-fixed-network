package valuehash

import (
	"encoding/hex"
)

func (dm *Dummy) unpack(s string) error {
	if b, err := hex.DecodeString(s); err != nil {
		return err
	} else {
		dm.b = b
	}

	return nil
}
