package hint

import (
	"github.com/spikeekips/mitum/util"
)

func (ht Hint) MarshalJSON() ([]byte, error) {
	var i interface{}
	if err := ht.Type().IsValid(nil); err == nil {
		i = ht.String()
	}

	return util.JSON.Marshal(i)
}

func (ht *Hint) UnmarshalJSON(b []byte) error {
	var s string
	if err := util.JSON.Unmarshal(b, &s); err != nil {
		return err
	} else if len(s) < 1 {
		return nil
	}

	if h, err := NewHintFromString(s); err != nil {
		return err
	} else {
		ht.t = h.t
		ht.version = h.version
	}

	return nil
}
