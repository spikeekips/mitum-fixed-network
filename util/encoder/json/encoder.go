package jsonenc

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/bluele/gcache"
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	JSONEncoderType = hint.Type("json-encoder")
	JSONEncoderHint = hint.NewHint(JSONEncoderType, "v0.0.1")
)

type Encoder struct {
	*hint.Hintset
	unpackers *hint.Hintmap
	cache     gcache.Cache
}

func NewEncoder() *Encoder {
	ca := gcache.New(100 * 100).LRU().Build()
	return &Encoder{
		Hintset:   hint.NewHintset(),
		unpackers: hint.NewHintmap(),
		cache:     ca,
	}
}

func (*Encoder) Hint() hint.Hint {
	return JSONEncoderHint
}

func (enc *Encoder) Add(ht hint.Hinter) error {
	if err := enc.Hintset.Add(ht); err != nil {
		return errors.Wrap(err, "JSONEncoder")
	}

	if err := enc.unpackers.Add(ht, enc.analyzeUnpack(ht)); err != nil {
		return errors.Wrap(err, "JSONEncoder")
	}

	return nil
}

func (*Encoder) Marshal(v interface{}) ([]byte, error) {
	return util.JSON.Marshal(v)
}

func (*Encoder) Unmarshal(b []byte, v interface{}) error {
	return util.JSON.Unmarshal(b, v)
}

func (enc *Encoder) Decode(b []byte) (hint.Hinter, error) {
	if isNil(b) {
		return nil, nil
	}

	ht, ub, err := enc.guessHint(b)
	if err != nil {
		return nil, err
	} else if err = ht.IsValid(nil); err != nil {
		return nil, nil // nolint:nilerr
	}

	i, err := enc.decode(ub, ht)
	if err != nil {
		return nil, errors.Wrap(err, "failed to json decode")
	}

	return i, nil
}

func (enc *Encoder) DecodeWithHint(b []byte, ht hint.Hint) (hint.Hinter, error) {
	if isNil(b) {
		return nil, nil
	}

	return enc.decode(b, ht)
}

func (enc *Encoder) DecodeSlice(b []byte) ([]hint.Hinter, error) {
	if isNil(b) {
		return nil, nil
	}

	var r []json.RawMessage
	if err := util.JSON.Unmarshal(b, &r); err != nil {
		return nil, errors.Wrap(err, "failed to decode slice")
	}

	s := make([]hint.Hinter, len(r))
	for i := range r {
		j, err := enc.Decode(r[i])
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode slice")
		}

		s[i] = j
	}

	return s, nil
}

func (enc *Encoder) DecodeMap(b []byte) (map[string]hint.Hinter, error) {
	if isNil(b) {
		return nil, nil
	}

	var r map[string]json.RawMessage
	if err := util.JSON.Unmarshal(b, &r); err != nil {
		return nil, errors.Wrap(err, "failed to decode slice")
	}

	s := map[string]hint.Hinter{}
	for i := range r {
		j, err := enc.Decode(r[i])
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode slice")
		}

		s[i] = j
	}

	return s, nil
}

func (enc *Encoder) decode(b []byte, ht hint.Hint) (hint.Hinter, error) {
	up, err := enc.findUnpacker(ht)
	if err != nil {
		return nil, fmt.Errorf("failed to find unpacker, %q: %w", ht, err)
	}

	i, err := up.F(b, ht)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode %T", up.Elem)
	} else if i == nil {
		return nil, nil
	}

	hinter, ok := i.(hint.Hinter)
	if !ok {
		return nil, util.WrongTypeError.Errorf("expected hint.Hinter, not %T", i)
	}

	return hinter, nil
}

func (enc *Encoder) guessHint(b []byte) (hint.Hint, []byte, error) {
	var ht hint.Hint
	switch i := bytes.TrimSpace(b); {
	case len(i) < 1:
		return ht, nil, nil
	default:
		return enc.extractHintFromObject(b)
	}
}

func (*Encoder) extractHintFromObject(b []byte) (hint.Hint, []byte, error) {
	var head HintedHead
	if err := util.JSON.Unmarshal(b, &head); err != nil {
		return head.H, nil, err
	} else if err = head.H.IsValid(nil); err != nil {
		return head.H, nil, nil // nolint:nilerr
	}

	return head.H, b, nil
}

func (enc *Encoder) findUnpacker(ht hint.Hint) (encoder.Unpacker, error) {
	i, err := enc.cache.Get(ht.RawString())
	if err == nil {
		switch t := i.(type) {
		case encoder.Unpacker:
			return t, nil
		case error:
			return encoder.Unpacker{}, t
		default:
			return encoder.Unpacker{}, util.WrongTypeError.Errorf("expected unpacker from cache, not %T", i)
		}
	}

	if !errors.Is(err, gcache.KeyNotFoundError) {
		return encoder.Unpacker{}, err
	}

	i, err = enc.unpackers.CompatibleByHint(ht)
	if err != nil {
		_ = enc.cache.Set(ht.RawString(), err)

		return encoder.Unpacker{}, err
	}

	j, ok := i.(encoder.Unpacker)
	if !ok {
		return encoder.Unpacker{}, util.WrongTypeError.Errorf("expected unpacker, not %T", i)
	}

	_ = enc.cache.Set(ht.RawString(), j)

	return j, nil
}

func (enc *Encoder) analyzeUnpack(ht hint.Hinter) encoder.Unpacker {
	ptr, elem := encoder.Ptr(ht)

	up := encoder.Unpacker{Elem: ht}

	switch ptr.Interface().(type) {
	case Unpackable:
		up.N = "JSONUnpackable"
		up.F = func(b []byte, ht hint.Hint) (interface{}, error) {
			i := reflect.New(elem.Type()).Interface()

			if err := i.(Unpackable).UnpackJSON(b, enc); err != nil {
				return nil, err
			}

			return reflect.ValueOf(i).Elem().Interface(), nil
		}
	case json.Unmarshaler:
		up.N = "JSONUnmarshaler"
		up.F = func(b []byte, _ hint.Hint) (interface{}, error) {
			i := reflect.New(elem.Type()).Interface()

			if err := i.(json.Unmarshaler).UnmarshalJSON(b); err != nil {
				return nil, err
			}

			return reflect.ValueOf(i).Elem().Interface(), nil
		}
	case encoding.TextUnmarshaler:
		up.N = "TextUnmarshaler"
		up.F = func(b []byte, _ hint.Hint) (interface{}, error) {
			i := reflect.New(elem.Type()).Interface()

			if err := i.(encoding.TextUnmarshaler).UnmarshalText(b); err != nil {
				return nil, err
			}

			return reflect.ValueOf(i).Elem().Interface(), nil
		}
	default:
		up.N = "default"
		up.F = func(b []byte, _ hint.Hint) (interface{}, error) {
			i := reflect.New(elem.Type()).Interface()

			if err := util.JSON.Unmarshal(b, i); err != nil {
				return nil, err
			}

			return reflect.ValueOf(i).Elem().Interface(), nil
		}
	}

	return encoder.AnalyzeSetHinter(up)
}

type Unpackable interface {
	UnpackJSON([]byte, *Encoder) error
}

func isNil(b []byte) bool {
	return len(b) < 1 || bytes.Equal(b, nullbytes)
}
