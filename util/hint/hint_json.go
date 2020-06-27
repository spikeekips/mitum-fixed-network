package hint

import (
	"github.com/spikeekips/mitum/util"
)

type hintJSON struct {
	Type    Type         `json:"type"`
	Version util.Version `json:"version"`
}

func (ht Hint) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(hintJSON{
		Type:    ht.t,
		Version: ht.version,
	})
}

func (ht *Hint) UnmarshalJSON(b []byte) error {
	var h hintJSON
	if err := util.JSON.Unmarshal(b, &h); err != nil {
		return err
	}

	ht.t = h.Type
	ht.version = h.Version

	return nil
}
