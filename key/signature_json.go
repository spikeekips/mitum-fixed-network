package key

import (
	"github.com/btcsuite/btcutil/base58"

	"github.com/spikeekips/mitum/util"
)

func (sg Signature) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(sg.String())
}

func (sg *Signature) UnmarshalJSON(b []byte) error {
	var s string
	if err := util.JSONUnmarshal(b, &s); err != nil {
		return err
	}

	*sg = Signature(base58.Decode(s))

	return nil
}
