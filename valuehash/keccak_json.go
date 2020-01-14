package valuehash

import (
	"github.com/btcsuite/btcutil/base58"

	"github.com/spikeekips/mitum/hint"
)

func (s512 SHA512) MarshalJSON() ([]byte, error) {
	return MarshalJSON(s512)
}

func (s512 SHA512) UnmarshalJSON(b []byte) error {
	h, err := UnmarshalJSON(b)
	if err != nil {
		return err
	}
	if s512.Hint().Type() != h.Hint.Type() {
		return hint.TypeDoesNotMatchError.Wrapf("a=%s b=%s", s512.Hint().Verbose(), h.Hint.Verbose())
	}

	copy(s512.b[:], base58.Decode(h.Hash))

	return nil
}
