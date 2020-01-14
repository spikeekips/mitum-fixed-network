package encoder

import (
	"github.com/spikeekips/mitum/errors"
	"github.com/spikeekips/mitum/hint"
)

var (
	InvalidEncoderFoundError = errors.NewError("invalid encoder found in encoders")
)

// Encoders is the collection of Encoder.
type Encoders struct {
	*hint.Hintset
	hinters *hint.Hintset
}

// NewEncoders returns new Encoders
func NewEncoders() *Encoders {
	return &Encoders{
		Hintset: hint.NewHintset(),
		hinters: hint.NewHintset(),
	}
}

// Add will get Encoder as argument.
func (es *Encoders) Add(ec *HintEncoder) error {
	if err := es.Hintset.Add(ec); err != nil {
		return err
	}

	ec.SetEncoders(es)

	return nil
}

// Encoder returns Encoder by Hint.
func (es *Encoders) HintEncoder(t hint.Type, version string) (*HintEncoder, error) {
	h, err := es.Hintset.Hinter(t, version)

	return h.(*HintEncoder), err
}

func (es *Encoders) AddHinter(target hint.Hinter) error {
	if err := es.hinters.Add(target); err != nil {
		return err
	}

	// analyze target by all encoders.
	for _, ecs := range es.Hinters() {
		for _, ec := range ecs {
			if _, err := (interface{})(ec).(*HintEncoder).Analyze(target); err != nil {
				return err
			}
		}
	}

	return nil
}

func (es *Encoders) Hinter(t hint.Type, version string) (hint.Hinter, error) {
	return es.hinters.Hinter(t, version)
}
