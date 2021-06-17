package bsonenc

import (
	"encoding"
	"reflect"

	"github.com/bluele/gcache"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"
)

var (
	BSONEncoderType = hint.Type("bson-encoder")
	BSONEncoderHint = hint.NewHint(BSONEncoderType, "v0.0.1")
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
	return BSONEncoderHint
}

func (enc *Encoder) Add(ht hint.Hinter) error {
	if err := enc.Hintset.Add(ht); err != nil {
		return xerrors.Errorf("BSONEncoder: %w", err)
	}

	if err := enc.unpackers.Add(ht, enc.analyzeUnpack(ht)); err != nil {
		return xerrors.Errorf("BSONEncoder: %w", err)
	}

	return nil
}

func (*Encoder) Marshal(v interface{}) ([]byte, error) {
	return bson.Marshal(v)
}

func (*Encoder) Unmarshal(b []byte, v interface{}) error {
	return bson.Unmarshal(b, v)
}

func (enc *Encoder) Decode(b []byte) (hint.Hinter, error) {
	ht, ub, err := enc.guessHint(b)
	if err != nil {
		return nil, err
	} else if err = ht.IsValid(nil); err != nil {
		return nil, nil // nolint:nilerr
	}

	i, err := enc.decode(ub, ht)
	if err != nil {
		return nil, xerrors.Errorf("failed to bson decode: %w", err)
	}

	return i, nil
}

func (enc *Encoder) DecodeWithHint(b []byte, ht hint.Hint) (hint.Hinter, error) {
	return enc.decode(b, ht)
}

func (enc *Encoder) DecodeSlice(b []byte) ([]hint.Hinter, error) {
	if len(b) < 1 {
		return nil, nil
	}
	raw := bson.Raw(b)

	r, err := raw.Values()
	if err != nil {
		return nil, err
	}

	s := make([]hint.Hinter, len(r))
	for i := range r {
		j, err := enc.Decode(r[i].Value)
		if err != nil {
			return nil, xerrors.Errorf("failed to decode slice: %w", err)
		}

		s[i] = j
	}

	return s, nil
}

func (enc *Encoder) DecodeMap(b []byte) (map[string]hint.Hinter, error) {
	var r map[string]bson.Raw
	if err := bson.Unmarshal(b, &r); err != nil {
		return nil, xerrors.Errorf("failed to decode slice: %w", err)
	}

	s := map[string]hint.Hinter{}
	for i := range r {
		j, err := enc.Decode(r[i])
		if err != nil {
			return nil, xerrors.Errorf("failed to decode slice: %w", err)
		}

		s[i] = j
	}

	return s, nil
}

func (enc *Encoder) decode(b []byte, ht hint.Hint) (hint.Hinter, error) {
	up, err := enc.findUnpacker(ht)
	if err != nil {
		return nil, xerrors.Errorf("failed to find unpacker: %w", err)
	}

	i, err := up.F(b)
	if err != nil {
		return nil, xerrors.Errorf("failed to decode %T: %w", up.Elem, err)
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
	ht, ub, err := enc.extractHintFromString(b)
	if err == nil {
		return ht, ub, nil
	} else if !xerrors.Is(err, util.NotFoundError) {
		return hint.Hint{}, nil, err
	}

	var head HintedHead
	if err := bson.Unmarshal(b, &head); err != nil {
		return hint.Hint{}, nil, err
	}

	return head.H, b, nil
}

func (*Encoder) extractHintFromString(b []byte) (hint.Hint, []byte, error) {
	var ht hint.Hint

	r := bson.RawValue{Type: bsontype.String, Value: b}
	s, _, ok := bsoncore.ReadString(r.Value)
	if !ok {
		return ht, nil, util.NotFoundError.Errorf("not string type")
	}

	hs, err := hint.ParseHintedString(s)
	if err != nil {
		return hs.Hint(), nil, err
	}

	return hs.Hint(), []byte(hs.Body()), nil
}

func (enc *Encoder) findUnpacker(ht hint.Hint) (encoder.Unpacker, error) {
	i, err := enc.cache.Get(ht.String())
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

	if !xerrors.Is(err, gcache.KeyNotFoundError) {
		return encoder.Unpacker{}, err
	}

	i, err = enc.unpackers.CompatibleByHint(ht)
	if err != nil {
		_ = enc.cache.Set(ht.String(), err)

		return encoder.Unpacker{}, err
	}

	j, ok := i.(encoder.Unpacker)
	if !ok {
		return encoder.Unpacker{}, util.WrongTypeError.Errorf("expected unpacker, not %T", i)
	}

	_ = enc.cache.Set(ht.String(), j)

	return j, nil
}

func (enc *Encoder) analyzeUnpack(ht hint.Hinter) encoder.Unpacker {
	ptr, elem := encoder.Ptr(ht)

	up := encoder.Unpacker{Elem: ht}

	switch ptr.Interface().(type) {
	case Unpackable:
		up.N = "BSONUnpackable"
		up.F = func(b []byte) (interface{}, error) {
			i := reflect.New(elem.Type()).Interface()

			if err := i.(Unpackable).UnpackBSON(b, enc); err != nil {
				return nil, err
			}

			return reflect.ValueOf(i).Elem().Interface(), nil
		}
	case bson.Unmarshaler:
		up.N = "BSONUnmarshaler"
		up.F = func(b []byte) (interface{}, error) {
			i := reflect.New(elem.Type()).Interface()

			if err := i.(bson.Unmarshaler).UnmarshalBSON(b); err != nil {
				return nil, err
			}

			return reflect.ValueOf(i).Elem().Interface(), nil
		}
	case encoding.TextUnmarshaler:
		up.N = "TextUnmarshaler"
		up.F = func(b []byte) (interface{}, error) {
			i := reflect.New(elem.Type()).Interface()

			if err := i.(encoding.TextUnmarshaler).UnmarshalText(b); err != nil {
				return nil, err
			}

			return reflect.ValueOf(i).Elem().Interface(), nil
		}
	default:
		up.N = "default"
		up.F = func(b []byte) (interface{}, error) {
			i := reflect.New(elem.Type()).Interface()

			if err := bson.Unmarshal(b, i); err != nil {
				return nil, err
			}

			return reflect.ValueOf(i).Elem().Interface(), nil
		}
	}

	return up
}

type Unpackable interface {
	UnpackBSON([]byte, *Encoder) error
}
