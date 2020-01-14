package mitum

import (
	"github.com/spikeekips/mitum/hint"
)

// Hintset is the collection of Encoder.
type Hintset struct {
	*hint.Hintset
}

// NewHintset returns new Hintset
func NewHintset() *Hintset {
	return &Hintset{
		Hintset: hint.NewHintset(),
	}
}

func (hs *Hintset) Add(i interface{}) error {
	if h, ok := i.(hint.Hinter); !ok {
		return hint.InvalidHInterTypeError.Wrapf("type=%T", i)
	} else if err := hs.Hintset.Add(h); err != nil {
		return err
	}

	return nil
}
