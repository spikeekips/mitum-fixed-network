package valuehash

import (
	"github.com/btcsuite/btcutil/base58"

	"github.com/spikeekips/mitum/hint"
)

func (s256 SHA256) MarshalJSON() ([]byte, error) {
	return MarshalJSON(s256)
}

func (s256 *SHA256) UnmarshalJSON(b []byte) error {
	h, err := UnmarshalJSON(b)
	if err != nil {
		return err
	}

	ht := h.JSONPackHintedHead.H
	if s256.Hint().Type() != ht.Type() {
		return hint.TypeDoesNotMatchError.Wrapf("a=%s b=%s", s256.Hint().Verbose(), ht.Verbose())
	}

	copy(s256.b[:], base58.Decode(h.Hash))

	return nil
}

func (s512 SHA512) MarshalJSON() ([]byte, error) {
	return MarshalJSON(s512)
}

func (s512 *SHA512) UnmarshalJSON(b []byte) error {
	h, err := UnmarshalJSON(b)
	if err != nil {
		return err
	}

	ht := h.JSONPackHintedHead.H
	if s512.Hint().Type() != ht.Type() {
		return hint.TypeDoesNotMatchError.Wrapf("a=%s b=%s", s512.Hint().Verbose(), ht.Verbose())
	}

	copy(s512.b[:], base58.Decode(h.Hash))

	return nil
}
