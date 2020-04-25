package valuehash

import (
	"github.com/btcsuite/btcutil/base58"
)

func (dm *Dummy) unpack(s string) error {
	copy(dm.b, base58.Decode(s))

	return nil
}
