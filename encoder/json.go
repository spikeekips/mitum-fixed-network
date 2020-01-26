package encoder

import (
	"encoding/json"
	"reflect"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
)

var jsonHint hint.Hint = hint.MustHint(hint.Type([2]byte{0x01, 0x01}), "0.1")

var jsoni = jsoniter.Config{
	EscapeHTML:             true,
	SortMapKeys:            false,
	ValidateJsonRawMessage: true,
}.Froze()

type JSONEncoder struct {
	cache   *cache
	hintset *hint.Hintset
}

func NewJSONEncoder() *JSONEncoder {
	return &JSONEncoder{
		cache: newCache(),
	}
}

func (je *JSONEncoder) SetHintset(hintset *hint.Hintset) {
	je.hintset = hintset
}

func (je JSONEncoder) Hint() hint.Hint {
	return jsonHint
}

func (je JSONEncoder) Marshal(i interface{}) ([]byte, error) {
	return json.Marshal(i)
}

func (je JSONEncoder) Unmarshal(b []byte, i interface{}) error {
	return json.Unmarshal(b, i)
}

func (je *JSONEncoder) Encode(i interface{}) ([]byte, error) {
	var target interface{} = i
	if i != nil {
		n, err := je.Pack(i)
		if err != nil {
			return nil, err
		}

		if n != nil {
			target = n
		}
	}

	return jsoni.Marshal(target)
}

func (je *JSONEncoder) Decode(b []byte, i interface{}) error {
	return je.Unpack(b, i)
}

func (je *JSONEncoder) DecodeByHint(b []byte) (hint.Hinter, error) {
	if je.hintset == nil {
		return nil, xerrors.Errorf("SetHintset() first")
	}

	h, err := je.loadHint(b)
	if err != nil {
		return nil, err
	}
	hinter, err := je.hintset.Hinter(h.Type(), h.Version())
	if err != nil {
		return nil, err
	}

	p := reflect.New(reflect.TypeOf(hinter))
	if err := je.Decode(b, p.Interface()); err != nil {
		return nil, err
	}

	return p.Elem().Interface().(hint.Hinter), nil
}

func (je *JSONEncoder) Analyze(i interface{}) error {
	_, elem := ExtractPtr(i)
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

func (je *JSONEncoder) analyze(i interface{}) (string, CachedPacker, error) { // nolint
	name, pack, unpack := je.analyzeInstance(i)

	// hint
	_, elem := ExtractPtr(i)
	if elem.Kind() == reflect.Struct {
		var hinter hint.Hinter
		if ht, ok := elem.Interface().(hint.Hinter); ok {
			hinter = ht
		}

		pack = je.wrapPackerHinter(hinter, pack)
		unpack = je.wrapUnpackerHinter(hinter, unpack)
	}

	return name, NewCachedPacker(elem.Type(), pack, unpack), nil
}

func (je *JSONEncoder) analyzeInstance(i interface{}) (string, jsonPackFunc, jsonUnpackFunc) { // nolint
	var names []string
	var pf jsonPackFunc
	var upf jsonUnpackFunc

	ptr, elem := ExtractPtr(i)

	if _, ok := elem.Interface().(JSONPackable); ok {
		names = append(names, "JSONPackable")
		pf = func(i interface{}) (interface{}, error) {
			return i.(JSONPackable).PackJSON(je)
		}
	}

	if _, ok := ptr.Interface().(JSONUnpackable); ok {
		names = append(names, "JSONUnpackable")
		upf = func(b []byte, i interface{}) (interface{}, error) {
			if err := i.(JSONUnpackable).UnpackJSON(b, je); err != nil {
				return nil, err
			}

			return reflect.ValueOf(i).Elem().Interface(), nil
		}
	}

	if pf != nil || upf != nil {
		if pf == nil {
			pf = je.packValueDefault
		}
		if upf == nil {
			upf = je.unpackValueDefault
		}

		return strings.Join(names, "+"), pf, upf
	}

	pf = je.packValueDefault
	upf = je.unpackValueDefault

	return encoderAnalyzedTypeDefault, pf, upf
}

func (je *JSONEncoder) Pack(i interface{}) (interface{}, error) {
	_, elem := ExtractPtr(i)

	if c, found := je.cache.Get(elem.Type()); found {
		if packer, ok := c.(CachedPacker); !ok {
			je.cache.Delete(elem.Type())
		} else if fn, ok := packer.Pack.(jsonPackFunc); !ok {
			je.cache.Delete(elem.Type())
		} else {
			return fn(i)
		}
	}

	_, cp, err := je.analyze(i)
	if err != nil {
		return nil, err
	}

	je.cache.Set(cp.Type, cp)

	return cp.Pack.(jsonPackFunc)(i)
}

func (je *JSONEncoder) packValueDefault(i interface{}) (interface{}, error) {
	return i, nil
}

func (je *JSONEncoder) Unpack(b []byte, i interface{}) error {
	if n, err := je.unpackValue(b, i); err != nil {
		return err
	} else if n != nil {
		reflect.ValueOf(i).Elem().Set(reflect.ValueOf(n))
		return nil
	}

	return jsoni.Unmarshal(b, i)
}

func (je *JSONEncoder) unpackValue(b []byte, i interface{}) (interface{}, error) {
	_, elem := ExtractPtr(i)

	if c, found := je.cache.Get(elem.Type()); found {
		if packer, ok := c.(CachedPacker); !ok {
			je.cache.Delete(elem.Type())
		} else if fn, ok := packer.Unpack.(jsonUnpackFunc); !ok {
			je.cache.Delete(elem.Type())
		} else {
			return je.callUnpacker(b, i, fn)
		}
	}

	_, cp, err := je.analyze(i)
	if err != nil {
		return nil, err
	}

	je.cache.Set(cp.Type, cp)

	return je.callUnpacker(b, i, cp.Unpack.(jsonUnpackFunc))
}

func (je *JSONEncoder) callUnpacker(b []byte, i interface{}, fn jsonUnpackFunc) (interface{}, error) {
	n, err := fn(b, i)
	if err != nil {
		return nil, err
	} else if n != nil {
		return n, nil
	}

	if err := jsoni.Unmarshal(b, i); err != nil {
		return nil, err
	}

	return reflect.ValueOf(i).Elem().Interface(), nil
}

func (je *JSONEncoder) unpackValueDefault(b []byte, i interface{}) (interface{}, error) {
	if err := jsoni.Unmarshal(b, i); err != nil {
		return nil, err
	}

	return reflect.ValueOf(i).Elem().Interface(), nil
}

func (je JSONEncoder) wrapPackerHinter(hinter hint.Hinter, fn jsonPackFunc) jsonPackFunc {
	if hinter == nil {
		return fn
	}

	return func(i interface{}) (interface{}, error) {
		o, err := fn(i)
		if err != nil {
			return nil, err
		}

		if _, ok := o.(IsJSONHinted); !ok {
			o = JSONPackHinted{H: hinter.Hint(), D: o}
		}

		return o, nil
	}
}

func (je JSONEncoder) wrapUnpackerHinter(hinter hint.Hinter, fn jsonUnpackFunc) jsonUnpackFunc {
	if hinter == nil {
		return fn
	}

	return func(b []byte, i interface{}) (interface{}, error) {
		var uj JSONUnpackHinted
		if err := jsoni.Unmarshal(b, &uj); err != nil {
			return nil, err
		}

		if err := hinter.Hint().IsCompatible(uj.H); err != nil {
			return nil, err
		}

		if uj.D != nil {
			return fn(uj.D, i)
		}

		return fn(b, i)
	}
}

func (je JSONEncoder) loadHint(b []byte) (hint.Hint, error) {
	var m JSONPackHintedHead
	if err := jsoni.Unmarshal(b, &m); err != nil {
		return hint.Hint{}, err
	}

	return m.H, nil
}

type jsonPackFunc func(interface{}) (interface{}, error)
type jsonUnpackFunc func([]byte, interface{}) (interface{}, error)

type JSONPackable interface {
	PackJSON(*JSONEncoder) (interface{}, error)
}

type JSONUnpackable interface {
	UnpackJSON([]byte, *JSONEncoder) error
}

type JSONPackHinted struct {
	H hint.Hint   `json:"_hint"`
	D interface{} `json:"_data"`
}

type JSONUnpackHinted struct {
	H hint.Hint       `json:"_hint"`
	D json.RawMessage `json:"_data,omitempty"`
}

type JSONPackHintedHead struct {
	H hint.Hint `json:"_hint"`
}

func NewJSONPackHintedHead(h hint.Hint) JSONPackHintedHead {
	return JSONPackHintedHead{H: h}
}

func (jh JSONPackHintedHead) IsJSONHinted() bool {
	return true
}

type IsJSONHinted interface {
	IsJSONHinted() bool
}
