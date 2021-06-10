package encoder

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/xerrors"
)

// Encoders is the collection of Encoder.
type Encoders struct {
	*hint.GlobalHintset
	hintset *hint.GlobalHintset
}

// NewEncoders returns new Encoders
func NewEncoders() *Encoders {
	return &Encoders{
		GlobalHintset: hint.NewGlobalHintset(),
		hintset:       hint.NewGlobalHintset(),
	}
}

func (es *Encoders) Initialize() error {
	if err := es.GlobalHintset.Initialize(); err != nil {
		return err
	}

	return es.hintset.Initialize()
}

// AddEncoder add Encoder.
func (es *Encoders) AddEncoder(ec Encoder) error {
	if !es.GlobalHintset.HasType(ec.Hint().Type()) {
		if err := es.GlobalHintset.AddType(ec.Hint().Type()); err != nil {
			return err
		}
	}

	if err := es.GlobalHintset.Add(ec); err != nil {
		return err
	}

	ec.SetHintset(es.hintset.Hintset)

	return nil
}

func (es *Encoders) Encoder(ty hint.Type, version string) (Encoder, error) {
	ht := hint.NewHint(ty, version)

	var hinter hint.Hinter
	if len(version) < 1 {
		i, err := es.GlobalHintset.Latest(ht.Type())
		if err != nil {
			return nil, err
		}
		hinter = i
	} else if i := es.GlobalHintset.Get(ht); i == nil {
		return nil, util.NotFoundError.Errorf("encoder, %q not found", ht)
	} else {
		hinter = i
	}

	i, ok := hinter.(Encoder)
	if !ok {
		return nil, xerrors.Errorf("not Encoder, %T", hinter)
	}
	return i, nil
}

func (es *Encoders) AddType(ty hint.Type) error {
	return es.hintset.AddType(ty)
}

func (es *Encoders) AddHinter(ht hint.Hinter) error {
	// analyze hinter by all encoders.
	hinters := es.Hintset.Hinters(ht.Hint().Type())
	for i := range hinters {
		enc := hinters[i]
		if err := (interface{})(enc).(Encoder).Analyze(ht); err != nil {
			return err
		}
	}

	return es.hintset.Add(ht)
}

func (es *Encoders) Compatible(t hint.Type, version string) (hint.Hinter, error) {
	return es.hintset.Compatible(hint.NewHint(t, version))
}
