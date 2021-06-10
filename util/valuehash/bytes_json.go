package valuehash

import "github.com/spikeekips/mitum/util"

func (hs Bytes) MarshalJSON() ([]byte, error) {
	return marshalJSON(hs)
}

func (hs *Bytes) UnmarshalJSON(b []byte) error {
	var s string
	if err := util.JSON.Unmarshal(b, &s); err != nil {
		return err
	}
	*hs = NewBytes(fromString(s))

	return nil
}
