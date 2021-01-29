package encoder

import (
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

type HintedString struct {
	h hint.Hint
	s string
}

func NewHintedString(ht hint.Hint, s string) HintedString {
	return HintedString{h: ht, s: s}
}

func (hs HintedString) Hint() hint.Hint {
	return hs.h
}

func (hs HintedString) IsEmpty() bool {
	return len(hs.s) < 1
}

func (hs HintedString) String() string {
	return hs.s
}

func (hs HintedString) IsValid([]byte) error {
	if err := hs.h.IsValid(nil); err != nil {
		return isvalid.InvalidError.Wrap(err)
	}

	if len(hs.s) < 1 {
		return isvalid.InvalidError.Errorf("empty string for HintedString")
	}

	return nil
}

func (hs HintedString) Encode(enc Encoder) (hint.Hinter, error) {
	return enc.DecodeWithHint(hs.h, []byte(hs.s))
}
