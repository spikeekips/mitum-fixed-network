package valuehash

import (
	"github.com/btcsuite/btcutil/base58"

	"github.com/spikeekips/mitum/hint"
)

func (dm Dummy) MarshalJSON() ([]byte, error) {
	return marshalJSON(dm)
}

func (dm *Dummy) UnmarshalJSON(b []byte) error {
	h, err := unmarshalJSON(b)
	if err != nil {
		return err
	}

	ht := h.JSONPackHintedHead.H
	if dm.Hint().Type() != ht.Type() {
		return hint.TypeDoesNotMatchError.Wrapf("a=%s b=%s", dm.Hint().Verbose(), ht.Verbose())
	}

	copy(dm.b, base58.Decode(h.Hash))

	return nil
}
