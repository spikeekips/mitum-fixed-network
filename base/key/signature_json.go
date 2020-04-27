package key

import (
	"github.com/btcsuite/btcutil/base58"

	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

func (sg Signature) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(sg.String())
}

func (sg *Signature) UnmarshalJSON(b []byte) error {
	var s string
	if err := jsonencoder.Unmarshal(b, &s); err != nil {
		return err
	}

	*sg = Signature(base58.Decode(s))

	return nil
}
