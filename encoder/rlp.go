package encoder

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
)

var rlpHint hint.Hint = hint.MustHint(hint.Type{0x01, 0x00}, "0.0.1")

type RLPEncoder struct {
	cache   *cache
	hintset *hint.Hintset
}

func NewRLPEncoder() *RLPEncoder {
	return &RLPEncoder{
		cache: newCache(),
	}
}

func (re *RLPEncoder) SetHintset(hintset *hint.Hintset) {
	re.hintset = hintset
}

func (re RLPEncoder) Hint() hint.Hint {
	return rlpHint
}

func (re RLPEncoder) Marshal(i interface{}) ([]byte, error) {
	return rlp.EncodeToBytes(i)
}

func (re RLPEncoder) Unmarshal(b []byte, i interface{}) error {
	return rlp.DecodeBytes(b, i)
}

func (re *RLPEncoder) Encode(i interface{}) ([]byte, error) {
	var target interface{} = i
	if i != nil {
		n, err := re.Pack(i)
		if err != nil {
			return nil, err
		}

		if n != nil {
			target = n
		}
	}

	var w bytes.Buffer
	if err := rlp.Encode(&w, target); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

func (re *RLPEncoder) Decode(b []byte, i interface{}) error {
	stream, _ := re.stream(b)

	return re.Unpack(stream, i)
}

func (re *RLPEncoder) DecodeByHint(b []byte) (hint.Hinter, error) {
	if re.hintset == nil {
		return nil, xerrors.Errorf("SetHintset() first")
	}

	h, err := re.loadHint(b)
	if err != nil {
		return nil, err
	}
	hinter, err := re.hintset.Hinter(h.Type(), h.Version())
	if err != nil {
		return nil, err
	}

	p := reflect.New(reflect.TypeOf(hinter))
	if err := re.Decode(b, p.Interface()); err != nil {
		return nil, err
	}

	return p.Elem().Interface().(hint.Hinter), nil
}

func (re *RLPEncoder) Analyze(i interface{}) error {
	_, elem := ExtractPtr(i)
	_, found := re.cache.Get(elem.Type())
	if found {
		return nil
	}

	_, cp, err := re.analyze(i)
	if err != nil {
		return err
	}

	re.cache.Set(cp.Type, cp)

	return nil
}

func (re *RLPEncoder) analyze(i interface{}) (string, CachedPacker, error) { // nolint
	name, pack, unpack := re.analyzeInstance(i)

	// hint
	_, elem := ExtractPtr(i)
	if elem.Kind() == reflect.Struct {
		var hinter hint.Hinter

		if ht, ok := elem.Interface().(hint.Hinter); ok {
			hinter = ht
		}

		pack = re.wrapPackerHinter(hinter, pack)
		unpack = re.wrapUnpackerHinter(hinter, unpack)
	}

	return name, NewCachedPacker(elem.Type(), pack, unpack), nil
}

func (re *RLPEncoder) analyzeInstance(i interface{}) (string, rlpPackFunc, rlpUnpackFunc) {
	var pf rlpPackFunc
	var upf rlpUnpackFunc

	_, elem := ExtractPtr(i)

	name, pf, upf := re.analyzeRLPPackable(i)
	if pf != nil && upf != nil {
		return name, pf, upf
	}

	switch elem.Interface().(type) {
	case int, int8, int16, int32, int64:
		return "packValueInt",
			re.withElemPacker(re.packValueInt),
			re.withPtrUnpacker(re.unpackValueInt)
	}

	switch elem.Type().Kind() {
	case reflect.Array, reflect.Slice:
		return "packValueArray",
			re.withElemPacker(re.packValueArray),
			re.withPtrUnpacker(re.unpackValueArray)
	case reflect.Map:
		return "packValueMap",
			re.withElemPacker(re.packValueMap),
			re.withPtrUnpacker(re.unpackValueMap)
	}

	pf = re.packValueDefault
	upf = re.unpackValueDefault

	return encoderAnalyzedTypeDefault, pf, upf
}

func (re *RLPEncoder) analyzeRLPPackable(i interface{}) (string, rlpPackFunc, rlpUnpackFunc) {
	var names []string
	var pf rlpPackFunc
	var upf rlpUnpackFunc

	ptr, elem := ExtractPtr(i)

	if _, ok := elem.Interface().(RLPPackable); ok {
		names = append(names, "RLPPackable")
		pf = func(i interface{}) (interface{}, error) {
			return i.(RLPPackable).PackRLP(re)
		}
	}

	if _, ok := ptr.Interface().(RLPUnpackable); ok {
		names = append(names, "RLPUnpackable")
		upf = func(reader, i interface{}) (interface{}, error) {
			stream, err := re.stream(reader)
			if err != nil {
				return nil, err
			}

			if err := i.(RLPUnpackable).UnpackRLP(stream, re); err != nil {
				return nil, err
			}

			return reflect.ValueOf(i).Elem().Interface(), nil
		}
	}

	if pf != nil || upf != nil {
		if pf == nil {
			pf = re.packValueDefault
		}
		if upf == nil {
			upf = re.unpackValueDefault
		}

		return strings.Join(names, "+"), pf, upf
	}

	return "", nil, nil
}

func (re *RLPEncoder) Pack(i interface{}) (interface{}, error) {
	return re.packValue(i)
}

func (re *RLPEncoder) packValue(i interface{}) (interface{}, error) {
	_, elem := ExtractPtr(i)

	if c, found := re.cache.Get(elem.Type()); found {
		if packer, ok := c.(CachedPacker); !ok {
			re.cache.Delete(elem.Type())
		} else if fn, ok := packer.Pack.(rlpPackFunc); !ok {
			re.cache.Delete(elem.Type())
		} else {
			return fn(i)
		}
	}

	_, cp, err := re.analyze(i)
	if err != nil {
		return nil, err
	}

	re.cache.Set(cp.Type, cp)

	return cp.Pack.(rlpPackFunc)(i)
}

func (re *RLPEncoder) packValueDefault(i interface{}) (interface{}, error) {
	return i, nil
}

func (re *RLPEncoder) packValueInt(i interface{}) (interface{}, error) {
	var o int64
	switch i.(type) {
	case int, int8, int16, int32, int64:
		switch t := i.(type) {
		case int:
			o = int64(t)
		case int8:
			o = int64(t)
		case int16:
			o = int64(t)
		case int32:
			o = int64(t)
		case int64:
			o = t
		}
	default:
		return nil, xerrors.Errorf("not int-like value: type=%T", i)
	}

	var w bytes.Buffer
	if err := binary.Write(&w, binary.LittleEndian, o); err != nil {
		return nil, err
	}

	return struct {
		B []byte
	}{
		B: w.Bytes(),
	}, nil
}

func (re *RLPEncoder) packValueArray(i interface{}) (interface{}, error) {
	elem := reflect.ValueOf(i)
	kind := elem.Type().Kind()
	if kind != reflect.Array && kind != reflect.Slice {
		return nil, xerrors.Errorf("not array|slice like value: type=%T", i)
	}

	var na []interface{}
	for i := 0; i < elem.Len(); i++ {
		n, err := re.packValue(elem.Index(i).Interface())
		if err != nil {
			return nil, err
		}

		na = append(na, n)
	}

	return na, nil
}

func (re *RLPEncoder) packValueMap(i interface{}) (interface{}, error) {
	elem := reflect.ValueOf(i)
	if elem.Kind() != reflect.Map {
		return nil, xerrors.Errorf("not map like value: type=%T", i)
	}

	na := make([][2]interface{}, len(elem.MapKeys())) // slice of (key, value)

	// sort key by alphanumeric order
	keys := make([]string, len(elem.MapKeys()))
	keyMap := map[string]reflect.Value{}

	for i, kv := range elem.MapKeys() {
		k := kv.Interface()

		var s string
		if v, ok := k.(fmt.Stringer); ok {
			s = v.String()
		} else {
			s = fmt.Sprintf("%v", k)
		}

		keys[i] = s
		keyMap[s] = kv
	}
	sort.Strings(keys)

	for i, k := range keys {
		kv := keyMap[k]
		vv := elem.MapIndex(kv)

		key, err := re.packValue(kv.Interface())
		if err != nil {
			return nil, err
		} else if key == nil {
			key = kv.Interface()
		}
		value, err := re.packValue(vv.Interface())
		if err != nil {
			return nil, err
		} else if value == nil {
			value = vv.Interface()
		}

		na[i] = [2]interface{}{key, value}
	}

	return na, nil
}

func (re *RLPEncoder) Unpack(reader interface{}, i interface{}) error {
	stream, err := re.stream(reader)
	if err != nil {
		return err
	}

	if n, err := re.unpackValue(stream, i); err != nil {
		return err
	} else if n != nil {
		reflect.ValueOf(i).Elem().Set(reflect.ValueOf(n))
		return nil
	}

	return stream.Decode(i)
}

func (re *RLPEncoder) unpackValue(reader interface{}, i interface{}) (interface{}, error) {
	_, elem := ExtractPtr(i)

	if c, found := re.cache.Get(elem.Type()); found {
		if packer, ok := c.(CachedPacker); !ok {
			re.cache.Delete(elem.Type())
		} else if fn, ok := packer.Unpack.(rlpUnpackFunc); !ok {
			re.cache.Delete(elem.Type())
		} else {
			return re.callUnpacker(reader, i, fn)
		}
	}

	_, cp, err := re.analyze(i)
	if err != nil {
		return nil, err
	}

	re.cache.Set(cp.Type, cp)

	return re.callUnpacker(reader, i, cp.Unpack.(rlpUnpackFunc))
}

func (re *RLPEncoder) callUnpacker(reader interface{}, i interface{}, fn rlpUnpackFunc) (interface{}, error) {
	n, err := fn(reader, i)
	if err != nil {
		return nil, err
	} else if n != nil {
		return n, nil
	}

	stream, err := re.stream(reader)
	if err != nil {
		return nil, err
	}

	if err := stream.Decode(i); err != nil {
		return nil, err
	}

	return reflect.ValueOf(i).Elem().Interface(), nil
}

func (re *RLPEncoder) unpackValueDefault(reader, i interface{}) (interface{}, error) {
	stream, err := re.stream(reader)
	if err != nil {
		return nil, err
	}

	if err := stream.Decode(i); err != nil {
		return nil, err
	}

	return reflect.ValueOf(i).Elem().Interface(), nil
}

func (re *RLPEncoder) unpackValueInt(reader interface{}, i interface{}) (interface{}, error) {
	switch i.(type) {
	case *int, *int8, *int16, *int32, *int64:
	default:
		return nil, xerrors.Errorf("not int-like target: type=%T", i)
	}

	stream, err := re.stream(reader)
	if err != nil {
		return nil, err
	}

	var ub struct {
		B []byte
	}

	if err := stream.Decode(&ub); err != nil {
		return nil, err
	}

	var o int64
	if err := binary.Read(bytes.NewBuffer(ub.B), binary.LittleEndian, &o); err != nil {
		return nil, err
	}

	var oo interface{} = o
	switch i.(type) {
	case *int:
		oo = int(o)
	case *int8:
		oo = int8(o)
	case *int16:
		oo = int16(o)
	case *int32:
		oo = int32(o)
	}

	return oo, nil
}

func (re *RLPEncoder) unpackValueArray(reader interface{}, i interface{}) (interface{}, error) {
	elem := reflect.ValueOf(i).Elem()
	kind := elem.Type().Kind()

	// var rv, elem reflect.Value
	if kind != reflect.Array && kind != reflect.Slice {
		return nil, xerrors.Errorf("not array target: %T", i)
	} else if kind == reflect.Array && elem.Len() < 1 {
		return elem.Interface(), nil
	}

	stream, err := re.stream(reader)
	if err != nil {
		return nil, err
	}

	var na []rlp.RawValue
	if err := stream.Decode(&na); err != nil {
		return nil, err
	} else if kind == reflect.Array && elem.Len() != len(na) {
		return nil, xerrors.Errorf("not match length of array; %d != %d", elem.Len(), len(na))
	}

	if len(na) < 1 {
		if kind == reflect.Slice {
			return reflect.MakeSlice(reflect.SliceOf(elem.Type().Elem()), 0, 0).Interface(), nil
		}

		return elem.Interface(), nil
	}

	var elemType reflect.Type
	if kind == reflect.Array {
		elemType = elem.Index(0).Type()
	} else {
		elemType = elem.Type().Elem()
	}

	rs := reflect.MakeSlice(reflect.SliceOf(elemType), 0, len(na))
	for i := 0; i < len(na); i++ {
		n, err := re.unpackValue(na[i], reflect.New(elemType).Interface())
		if err != nil {
			return nil, err
		}

		rs = reflect.Append(rs, reflect.ValueOf(n))
	}

	if kind == reflect.Slice {
		return rs.Interface(), nil
	}

	for i := 0; i < rs.Len(); i++ {
		elem.Index(i).Set(rs.Index(i))
	}

	return elem.Interface(), nil
}

func (re *RLPEncoder) unpackValueMap(reader interface{}, i interface{}) (interface{}, error) {
	elem := reflect.ValueOf(i).Elem()
	if elem.Type().Kind() != reflect.Map {
		return nil, xerrors.Errorf("not map target: %T", i)
	}

	stream, err := re.stream(reader)
	if err != nil {
		return nil, err
	}

	var na []rlp.RawValue
	if err := stream.Decode(&na); err != nil {
		return nil, err
	}

	m := reflect.MakeMapWithSize(
		reflect.MapOf(elem.Type().Key(), elem.Type().Elem()),
		0,
	)
	if len(na) < 1 {
		return m.Interface(), nil
	}

	for _, a := range na {
		var e [2]rlp.RawValue
		if err := rlp.DecodeBytes(a, &e); err != nil {
			return nil, err
		}

		var key, value reflect.Value
		rk := reflect.New(elem.Type().Key())
		if n, err := re.unpackValue(e[0], rk.Interface()); err != nil {
			return nil, err
		} else {
			key = reflect.ValueOf(n)
		}

		rv := reflect.New(elem.Type().Elem())
		if n, err := re.unpackValue(e[1], rv.Interface()); err != nil {
			return nil, err
		} else {
			value = reflect.ValueOf(n)
		}

		m.SetMapIndex(key, value)
	}

	return m.Interface(), nil
}

func (re RLPEncoder) stream(reader interface{}) (*rlp.Stream, error) {
	var b []byte
	switch t := reader.(type) {
	case *rlp.Stream:
		return t, nil
	case rlp.RawValue:
		b = t
	case []byte:
		b = t
	default:
		return nil, xerrors.Errorf("unacceptable reader found: %T", reader)
	}

	return rlp.NewStream(bytes.NewReader(b), uint64(len(b))), nil
}

func (re RLPEncoder) withElemPacker(fn rlpPackFunc) rlpPackFunc {
	return func(i interface{}) (interface{}, error) {
		_, elem := ExtractPtr(i)
		return fn(elem.Interface())
	}
}

func (re RLPEncoder) withPtrUnpacker(fn rlpUnpackFunc) rlpUnpackFunc {
	return func(reader, i interface{}) (interface{}, error) {
		ptr, _ := ExtractPtr(i)
		return fn(reader, ptr.Interface())
	}
}

func (re RLPEncoder) wrapPackerHinter(hinter hint.Hinter, fn rlpPackFunc) rlpPackFunc {
	if hinter == nil {
		return fn
	}

	return func(i interface{}) (interface{}, error) {
		v, err := fn(i)
		if err != nil {
			return nil, err
		}

		return NewRLPPaackHinted(hinter.Hint(), v), nil
	}
}

func (re RLPEncoder) wrapUnpackerHinter(hinter hint.Hinter, fn rlpUnpackFunc) rlpUnpackFunc {
	if hinter == nil {
		return fn
	}

	return func(reader, i interface{}) (interface{}, error) {
		stream, err := re.stream(reader)
		if err != nil {
			return nil, err
		}

		var rh rlpUnpackHinted
		if err := stream.Decode(&rh); err != nil {
			return nil, err
		}

		if err := hinter.Hint().IsCompatible(rh.H); err != nil {
			return nil, err
		}

		return fn(rh.B, i)
	}
}

func (re RLPEncoder) loadHint(reader interface{}) (hint.Hint, error) {
	stream, err := re.stream(reader)
	if err != nil {
		return hint.Hint{}, err
	}

	var rh rlpUnpackHinted
	if err := stream.Decode(&rh); err != nil {
		return hint.Hint{}, err
	}

	return rh.H, nil
}

type (
	rlpPackFunc   func(interface{}) (interface{}, error)
	rlpUnpackFunc func(interface{}, interface{}) (interface{}, error)
)

type RLPPackable interface {
	PackRLP(*RLPEncoder) (interface{}, error)
}

type RLPUnpackable interface {
	UnpackRLP(*rlp.Stream, *RLPEncoder) error
}

type rlpPackHinted struct {
	H hint.Hint
	B interface{}
}

type rlpUnpackHinted struct {
	H hint.Hint
	B rlp.RawValue
}

func NewRLPPaackHinted(h hint.Hint, b interface{}) rlpPackHinted {
	return rlpPackHinted{H: h, B: b}
}
