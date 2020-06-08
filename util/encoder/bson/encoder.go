package bsonenc

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	BSONType = hint.MustNewType(0x01, 0x02, "bson")
	bsonHint = hint.MustHint(BSONType, "0.0.1")
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

func (be *Encoder) SetHintset(hintset *hint.Hintset) {
	be.hintset = hintset
}

func (be Encoder) Hint() hint.Hint {
	return bsonHint
}

func (be Encoder) Marshal(i interface{}) ([]byte, error) {
	return Marshal(i)
}

func (be Encoder) Unmarshal(b []byte, i interface{}) error {
	return Unmarshal(b, i)
}

func (be *Encoder) Decode(b []byte, i interface{}) error {
	return be.Unpack(b, i)
}

func (be *Encoder) DecodeByHint(b []byte) (hint.Hinter, error) {
	if be.hintset == nil {
		return nil, xerrors.Errorf("SetHintset() first")
	}

	h, err := be.loadHint(b)
	if err != nil {
		return nil, err
	}

	hinter, err := be.hintset.Hinter(h.Type(), h.Version())
	if err != nil {
		return nil, err
	}

	p := reflect.New(reflect.TypeOf(hinter))
	if err := be.Decode(b, p.Interface()); err != nil {
		return nil, err
	}

	return p.Elem().Interface().(hint.Hinter), nil
}

func (be *Encoder) Analyze(i interface{}) error {
	_, elem := encoder.ExtractPtr(i)
	_, found := be.cache.Get(elem.Type())
	if found {
		return nil
	}

	_, cp, err := be.analyze(i)
	if err != nil {
		return err
	}

	be.cache.Set(cp.Type, cp)

	return nil
}

func (be *Encoder) analyze(i interface{}) (string, encoder.CachedPacker, error) { // nolint
	name, unpack := be.analyzeInstance(i)

	_, elem := encoder.ExtractPtr(i)

	return name, encoder.NewCachedPacker(elem.Type(), unpack), nil
}

func (be *Encoder) analyzeInstance(i interface{}) (string, unpackFunc) {
	var name string
	var upf unpackFunc

	ptr, _ := encoder.ExtractPtr(i)

	if _, ok := ptr.Interface().(bson.Unmarshaler); ok {
		name = "BSONUnmarshaler"
		upf = func(b []byte, i interface{}) error {
			return i.(bson.Unmarshaler).UnmarshalBSON(b)
		}
	} else if _, ok := ptr.Interface().(Unpackable); ok {
		name = "Unpackable"
		upf = func(b []byte, i interface{}) error {
			return i.(Unpackable).UnpackBSON(b, be)
		}
	}

	if upf == nil {
		name = encoder.EncoderAnalyzedTypeDefault
		upf = be.unpackValueDefault
	}

	return name, upf
}

func (be *Encoder) Unpack(b []byte, i interface{}) error {
	_, elem := encoder.ExtractPtr(i)

	if c, found := be.cache.Get(elem.Type()); found {
		if packer, ok := c.(encoder.CachedPacker); !ok {
			be.cache.Delete(elem.Type())
		} else if fn, ok := packer.Unpack.(unpackFunc); !ok {
			be.cache.Delete(elem.Type())
		} else {
			return fn(b, i)
		}
	}

	_, cp, err := be.analyze(i)
	if err != nil {
		return err
	}

	be.cache.Set(cp.Type, cp)

	return cp.Unpack.(unpackFunc)(b, i)
}

func (be *Encoder) unpackValueDefault(b []byte, i interface{}) error {
	return Unmarshal(b, i)
}

func (be Encoder) loadHint(b []byte) (hint.Hint, error) {
	var o PackHintedHead
	if err := Unmarshal(b, &o); err != nil {
		return hint.Hint{}, err
	}

	return o.H, nil
}

type unpackFunc func([]byte, interface{}) error

type Unpackable interface {
	UnpackBSON([]byte, *Encoder) error
}
