package encoder

import (
	"github.com/spikeekips/mitum/hint"
)

// Encoders is the collection of Encoder.
type Encoders struct {
	*hint.Hintset
	hintset *hint.Hintset
}

// NewEncoders returns new Encoders
func NewEncoders() *Encoders {
	return &Encoders{
		Hintset: hint.NewHintset(),
		hintset: hint.NewHintset(),
	}
}

// AddEncoder add Encoder.
func (es *Encoders) AddEncoder(ec Encoder) error {
	if err := es.Hintset.Add(ec); err != nil {
		return err
	}

	ec.SetHintset(es.hintset)

	return nil
}

// Encoder returns Encoder by Hint.
func (es *Encoders) Encoder(t hint.Type, version hint.Version) (Encoder, error) {
	h, err := es.Hintset.Hinter(t, version)

	return h.(Encoder), err
}

func (es *Encoders) AddHinter(hinter hint.Hinter) error {
	// analyze hinter by all encoders.
	for _, ecs := range es.Hintset.Hinters() {
		for _, ec := range ecs {
			if err := (interface{})(ec).(Encoder).Analyze(hinter); err != nil {
				return err
			}
		}
	}

	if err := es.hintset.Add(hinter); err != nil {
		return err
	}

	return nil
}

func (es *Encoders) Hinter(t hint.Type, version hint.Version) (hint.Hinter, error) {
	return es.hintset.Hinter(t, version)
}
