package encoder

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
)

func (hs *HintedString) UnmarshalJSON(b []byte) error {
	var s string
	if err := util.JSON.Unmarshal(b, &s); err != nil {
		return err
	} else if len(s) < 1 {
		return nil
	}

	h, us, err := hint.ParseHintedString(s)
	if err != nil {
		return err
	}

	hs.h = h
	hs.s = us

	return nil
}
