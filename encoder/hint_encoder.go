package encoder

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
)

type HintEncoder struct {
	enc      Encoder
	encs     *Encoders
	encoders *sync.Map //  caching for EncodeHint
}

func NewHintEncoder(enc Encoder) *HintEncoder {
	return &HintEncoder{
		enc:      enc,
		encoders: &sync.Map{},
	}
}

func (he *HintEncoder) Hint() hint.Hint {
	return he.enc.Hint()
}

func (he *HintEncoder) Encoder() Encoder {
	return he.enc
}

func (he *HintEncoder) SetEncoders(encs *Encoders) {
	he.encs = encs
}

func (he *HintEncoder) Analyze(target interface{}) (EncoderHint, error) {
	if target == nil {
		return EncoderHint{}, xerrors.Errorf("nil target")
	}

	_, isHinter := target.(hint.Hinter)

	var encode func(*HintEncoder, interface{}) ([]byte, error)
	var decode func(*HintEncoder, []byte, interface{}) error

	ins, ptr := getPtr(target)
	_, hasEncoder := he.Encoder().Encoder(ins)
	_, hasDecoder := he.Encoder().Decoder(ptr)

	if hasEncoder {
		encode = func(enc *HintEncoder, target interface{}) ([]byte, error) {
			fn, _ := enc.Encoder().Encoder(target)
			encoded, err := fn(enc)
			if err != nil {
				return nil, err
			}

			hinter, _ := target.(hint.Hinter)
			return enc.Encoder().EncodeHint(hinter, encoded)
		}
	} else {
		encode = func(enc *HintEncoder, target interface{}) ([]byte, error) {
			hinter, _ := target.(hint.Hinter)
			return enc.Encoder().EncodeHint(hinter, target)
		}
	}

	if hasDecoder {
		decode = func(enc *HintEncoder, b []byte, target interface{}) error {
			fn, _ := enc.Encoder().Decoder(target)
			return fn(enc, b)
		}
	} else {
		decode = func(enc *HintEncoder, b []byte, target interface{}) error {
			return enc.Encoder().Unmarshal(b, target)
		}
	}

	dh := NewEncoderHint(encode, decode, hasEncoder, hasDecoder, isHinter)
	he.addEncoder(ptr, dh)

	return dh, nil
}

func (he *HintEncoder) encoder(target interface{}) (EncoderHint, bool) {
	_, ptr := getPtr(target)
	i, found := he.encoders.Load(reflect.TypeOf(ptr))
	if !found {
		return EncoderHint{}, false
	}

	return i.(EncoderHint), true
}

func (he *HintEncoder) addEncoder(target interface{}, dh EncoderHint) {
	t := reflect.TypeOf(target)
	if len(t.Name()) < 1 {
		return
	}

	he.encoders.Store(reflect.TypeOf(target), dh)
}

// loadHint read Hint from input []byte and check the loaded Hint is compatible
// with target Hint. The target Hint should be compatible; same type and same
// version or compatible version.
func (he *HintEncoder) loadHint(b []byte, target hint.Hint) (hint.Hint, error) {
	ht, err := he.Encoder().DecodeHint(b)
	if err != nil {
		return hint.Hint{}, err
	}

	if err := target.IsCompatible(ht); err != nil {
		return hint.Hint{}, err
	}

	return target, nil
}

func (he *HintEncoder) Encode(target interface{}) ([]byte, error) {
	if dh, found := he.encoder(target); found {
		return dh.Encode(he, target)
	}

	dh, err := he.Analyze(target)
	if err != nil {
		return nil, err
	}
	he.addEncoder(target, dh)

	return dh.Encode(he, target)
}

func (he *HintEncoder) Decode(b []byte, target interface{}) error {
	if target == nil {
		return xerrors.Errorf("target is nil")
	} else if reflect.ValueOf(target).Kind() != reflect.Ptr {
		return xerrors.Errorf("target is not ptr")
	}

	if h, isHinter := target.(hint.Hinter); isHinter {
		if _, err := he.loadHint(b, h.Hint()); err != nil {
			return err
		}
	}

	if dh, found := he.encoder(target); found {
		return dh.Decode(he, b, target)
	}

	dh, err := he.Analyze(target)
	if err != nil {
		return err
	}
	he.addEncoder(target, dh)

	return dh.Decode(he, b, target)
}

func (he *HintEncoder) DecodeByHint(b []byte) (interface{}, error) {
	if he.encs == nil {
		return nil, xerrors.Errorf("he.encs is nil, HintEncoder.SetEncoders() must be called.")
	}

	var hinter hint.Hinter
	if ht, err := he.Encoder().DecodeHint(b); err != nil {
		return nil, err
	} else if h, err := he.encs.Hinter(ht.Type(), ht.Version()); err != nil {
		return nil, err
	} else {
		hinter = h
	}

	target := reflect.New(reflect.TypeOf((interface{})(hinter))).Interface()
	if dh, found := he.encoder(hinter); found {
		if err := dh.Decode(he, b, target); err != nil {
			return nil, err
		}

		return reflect.ValueOf(target).Elem().Interface(), nil
	}

	if dh, err := he.Analyze(target); err != nil {
		return nil, err
	} else {
		he.addEncoder(target, dh)

		if err := dh.Decode(he, b, target); err != nil {
			return nil, err
		}
	}
	return reflect.ValueOf(target).Elem().Interface(), nil
}

type EncoderHint struct {
	IsHinter   bool
	HasEncoder bool
	HasDecoder bool
	encode     func(*HintEncoder, interface{}) ([]byte, error)
	decode     func(*HintEncoder, []byte, interface{}) error
}

func NewEncoderHint(
	encode func(*HintEncoder, interface{}) ([]byte, error),
	decode func(*HintEncoder, []byte, interface{}) error,
	hasEncoder,
	hasDecoder,
	isHinter bool,
) EncoderHint {
	return EncoderHint{
		IsHinter:   isHinter,
		HasEncoder: hasEncoder,
		HasDecoder: hasDecoder,
		encode:     encode,
		decode:     decode,
	}
}

func (dh EncoderHint) String() string {
	b, _ := json.Marshal(map[string]interface{}{
		"is_hinter":   dh.IsHinter,
		"has_encoder": dh.HasEncoder,
		"has_decoder": dh.HasDecoder,
		"encode":      fmt.Sprintf("%T", dh.encode),
		"decode":      fmt.Sprintf("%T", dh.decode),
	})

	return string(b)
}

func (dh EncoderHint) Encode(enc *HintEncoder, target interface{}) ([]byte, error) {
	return dh.encode(enc, target)
}

func (dh EncoderHint) Decode(enc *HintEncoder, b []byte, target interface{}) error {
	return dh.decode(enc, b, target)
}

func getPtr(i interface{}) (interface{}, interface{}) {
	var ins interface{} = i
	var ptr interface{} = i
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		ins = reflect.ValueOf(i).Elem().Interface()
	} else {
		ptr = reflect.New(reflect.TypeOf(i)).Interface()
	}

	return ins, ptr
}
