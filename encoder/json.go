package encoder

import (
	"encoding/json"
	"reflect"

	jsoniter "github.com/json-iterator/go"

	"github.com/spikeekips/mitum/hint"
)

var jsoni = jsoniter.Config{
	EscapeHTML:             true,
	SortMapKeys:            false,
	ValidateJsonRawMessage: true,
}.Froze()

var jsonHint hint.Hint = hint.MustHint(hint.Type([2]byte{0x01, 0x00}), "0.1")

func NewJSONHintEncoder() *HintEncoder {
	return NewHintEncoder(JSON{})
}

type JSON struct {
}

func (je JSON) Hint() hint.Hint {
	return jsonHint
}

func (je JSON) Marshal(v interface{}) ([]byte, error) {
	return jsoni.Marshal(v)
}

func (je JSON) Unmarshal(b []byte, v interface{}) error {
	return jsoni.Unmarshal(b, v)
}

func (je JSON) Encoder(target interface{}) (func(*HintEncoder) (interface{}, error), bool) {
	fn, ok := target.(JSONEncoder)
	if !ok {
		return nil, false
	}

	return fn.EncodeJSON, true
}

func (je JSON) Decoder(target interface{}) (func(*HintEncoder, []byte) error, bool) {
	fn, ok := target.(JSONDecoder)
	if !ok {
		return nil, false
	}

	return fn.DecodeJSON, true
}

func (je JSON) DecodeHint(b []byte) (hint.Hint, error) {
	var h JSONHinterHead
	if err := jsoni.Unmarshal(b, &h); err != nil {
		return hint.Hint{}, err
	}

	if err := h.H.IsValid(); err != nil {
		return hint.Hint{}, err
	}

	return h.H, nil
}

// Encode encodes input data with Hint information.
func (je JSON) EncodeHint(hinter hint.Hinter, encoded interface{}) ([]byte, error) {
	switch t := encoded.(type) {
	case json.RawMessage:
		return t, nil
	case []byte:
		return t, nil
	case map[string]interface{}:
		if hinter != nil {
			return je.Marshal(encoded)
		}

		if _, found := t["_hint"]; !found {
			t["_hint"] = hinter.Hint()
			return je.Marshal(t)
		}
	}

	if hinter == nil {
		return je.Marshal(encoded)
	}

	kind := reflect.TypeOf(encoded).Kind()
	if kind != reflect.Ptr {
		return je.Marshal(encoded)
	}

	ptr := reflect.ValueOf(encoded)
	for i := 0; i < ptr.Elem().NumField(); i++ {
		f := ptr.Elem().Field(i)

		if f.Type() != reflect.TypeOf(JSONHinterHead{}) {
			continue
		}

		f.Set(reflect.ValueOf(NewJSONHinterHead(hinter.Hint())))
	}

	return je.Marshal(encoded)
}

type JSONHinterHead struct {
	H hint.Hint `json:"_hint"`
}

func NewJSONHinterHead(ht hint.Hint) JSONHinterHead {
	return JSONHinterHead{H: ht}
}

type JSONEncoder interface {
	EncodeJSON(*HintEncoder) (interface{}, error)
}

type JSONDecoder interface {
	DecodeJSON(*HintEncoder, []byte) error
}
