package key

import (
	"github.com/spikeekips/mitum/util"
)

func (kd *KeyDecoder) UnmarshalJSON(b []byte) error {
	var s string
	if err := util.JSON.Unmarshal(b, &s); err != nil {
		return err
	}

	if h, us, err := ParseString(s); err != nil {
		return err
	} else {
		kd.h = h
		kd.s = us
	}

	return nil
}
