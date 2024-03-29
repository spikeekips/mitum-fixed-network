package state

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	HintedValueType   = hint.Type("state-hinted-value")
	HintedValueHint   = hint.NewHint(HintedValueType, "v0.0.1")
	HintedValueHinter = HintedValue{BaseHinter: hint.NewBaseHinter(HintedValueHint)}
)

type HintedValue struct {
	hint.BaseHinter
	v hint.Hinter
}

func NewHintedValue(v hint.Hinter) (HintedValue, error) {
	hv := HintedValue{BaseHinter: hint.NewBaseHinter(HintedValueHint)}
	nhv, err := hv.Set(v)
	if err != nil {
		return HintedValue{}, err
	}

	return nhv.(HintedValue), nil
}

func (hv HintedValue) IsValid([]byte) error {
	if err := hv.BaseHinter.IsValid(nil); err != nil {
		return err
	}

	if is, ok := hv.v.(isvalid.IsValider); ok {
		if err := is.IsValid(nil); err != nil {
			return err
		}
	}

	return nil
}

func (hv HintedValue) Hash() valuehash.Hash {
	return hv.v.(valuehash.Hasher).Hash()
}

func (hv HintedValue) Interface() interface{} {
	return hv.v
}

func (hv HintedValue) Equal(v Value) bool {
	return hv.Hash().Equal(v.Hash())
}

func (hv HintedValue) Set(v interface{}) (Value, error) {
	if _, ok := v.(hint.Hinter); !ok {
		return nil, util.WrongTypeError.Errorf("not Hinter: %T", v)
	} else if _, ok := v.(valuehash.Hasher); !ok {
		return nil, util.WrongTypeError.Errorf("not valuehash.Hasher: %T", v)
	}

	hv.v = v.(hint.Hinter)

	return hv, nil
}
