package key

import (
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
)

func (sg Signature) MarshalJSON() ([]byte, error) {
	return json.Marshal(sg.String())
}

func (sg *Signature) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	*sg = Signature(base58.Decode(s))

	return nil
}
