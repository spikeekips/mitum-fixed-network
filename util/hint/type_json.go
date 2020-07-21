package hint

import (
	"encoding/hex"

	"github.com/spikeekips/mitum/util"
)

func (ty Type) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(hex.EncodeToString(ty.Bytes()))
}

func (ty *Type) UnmarshalJSON(b []byte) error {
	var s string
	if err := util.JSON.Unmarshal(b, &s); err != nil {
		return err
	}

	return ty.unpack(s)
}
