package jsonenc

import (
	"encoding/json"
	"reflect"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	jsonNULL       []byte = []byte("null")
	jsonNULLLength int    = len(jsonNULL)
	JSONType              = hint.MustNewType(0x01, 0x01, "json")
	jsonHint              = hint.MustHint(JSONType, "0.0.1")
)

type Encoder struct {
	cache   *encoder.Cache
	hintset *hint.Hintset
}

func NewEncoder() *Encoder {
	return &Encoder{
		cache: encoder.NewCache(),
	}
}

func (je *Encoder) SetHintset(hintset *hint.Hintset) {
	je.hintset = hintset
}

func (je Encoder) Hint() hint.Hint {
	return jsonHint
}

func (je Encoder) Marshal(i interface{}) ([]byte, error) {
	return Marshal(i)
}

func (je Encoder) Unmarshal(b []byte, i interface{}) error {
	return Unmarshal(b, i)
}

func (je *Encoder) Decode(b []byte, i interface{}) error {
	return je.Unpack(b, i)
}

func (je *Encoder) DecodeByHint(b []byte) (hint.Hinter, error) {
	if isNullRawMesage(b) {
		return nil, nil
	}

	if je.hintset == nil {
		return nil, xerrors.Errorf("SetHintset() first: %q", string(b))
	}

	h, err := je.loadHint(b)
	if err != nil {
		return nil, err
	}
	hinter, err := je.hintset.Hinter(h.Type(), h.Version())
	if err != nil {
		return nil, xerrors.Errorf(`failed to find hinter: hint=%s input="%s": %w`, h.Verbose(), string(b), err)
	}

	p := reflect.New(reflect.TypeOf(hinter))
	if err := je.Decode(b, p.Interface()); err != nil {
		return nil, err
	}

	return p.Elem().Interface().(hint.Hinter), nil
}

func (je *Encoder) Analyze(i interface{}) error {
	_, elem := encoder.ExtractPtr(i)
	_, found := je.cache.Get(elem.Type())
	if found {
		return nil
	}

	_, cp, err := je.analyze(i)
	if err != nil {
		return err
	}

	je.cache.Set(cp.Type, cp)

	return nil
}

func (je *Encoder) analyze(i interface{}) (string, encoder.CachedPacker, error) { // nolint
	name, unpack := je.analyzeInstance(i)

	_, elem := encoder.ExtractPtr(i)

	return name, encoder.NewCachedPacker(elem.Type(), unpack), nil
}

func (je *Encoder) analyzeInstance(i interface{}) (string, unpackFunc) {
	ptr, _ := encoder.ExtractPtr(i)

	if _, ok := ptr.Interface().(json.Unmarshaler); ok {
		return "JSONUnmarshaler", func(b []byte, i interface{}) error {
			return i.(json.Unmarshaler).UnmarshalJSON(b)
		}
	} else if _, ok := ptr.Interface().(Unpackable); ok {
		return "JSONUnpackable", func(b []byte, i interface{}) error {
			return i.(Unpackable).UnpackJSON(b, je)
		}
	}

	return encoder.EncoderAnalyzedTypeDefault, je.unpackValueDefault
}

func (je *Encoder) Unpack(b []byte, i interface{}) error {
	_, elem := encoder.ExtractPtr(i)

	if c, found := je.cache.Get(elem.Type()); found {
		if packer, ok := c.(encoder.CachedPacker); !ok {
			je.cache.Delete(elem.Type())
		} else if fn, ok := packer.Unpack.(unpackFunc); !ok {
			je.cache.Delete(elem.Type())
		} else {
			return fn(b, i)
		}
	}

	_, cp, err := je.analyze(i)
	if err != nil {
		return err
	}

	je.cache.Set(cp.Type, cp)

	return cp.Unpack.(unpackFunc)(b, i)
}

func (je *Encoder) unpackValueDefault(b []byte, i interface{}) error {
	return Unmarshal(b, i)
}

func (je Encoder) loadHint(b []byte) (hint.Hint, error) {
	var m HintedHead
	if err := Unmarshal(b, &m); err != nil {
		return hint.Hint{}, err
	}

	return m.H, nil
}

type (
	unpackFunc func([]byte, interface{}) error
)

type Unpackable interface {
	UnpackJSON([]byte, *Encoder) error
}
