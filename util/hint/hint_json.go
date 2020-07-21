package hint

import (
	"github.com/spikeekips/mitum/util"
)

func (ht Hint) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(ht.String())
}

func (ht *Hint) UnmarshalJSON(b []byte) error {
	var s string
	if err := util.JSON.Unmarshal(b, &s); err != nil {
		return err
	}

	if h, err := NewHintFromString(s); err != nil {
		return err
	} else {
		ht.t = h.t
		ht.version = h.version
	}

	return nil
}
