package encoder

import (
	"github.com/spikeekips/mitum/util/hint"
)

type HintedString struct {
	hint.HintedString
}

func NewHintedString(ht hint.Hint, s string) HintedString {
	return HintedString{HintedString: hint.NewHintedString(ht, s)}
}

func (hs HintedString) Decode(enc Encoder) (hint.Hinter, error) {
	return enc.DecodeWithHint([]byte(hs.Body()), hs.Hint())
}
