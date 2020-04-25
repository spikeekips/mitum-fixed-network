package encoder

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
)

func (s0 sup0) MarshalBSON() ([]byte, error) {
	return bson.Marshal(struct {
		A string
	}{
		A: s0.A + "-packed",
	})
}

func (s0 *sup0) UnpackBSON(b []byte, _ *BSONEncoder) error {
	var us sup0
	if err := bson.Unmarshal(b, &us); err != nil {
		return err
	}

	s0.A = us.A + "-unpacked"

	return nil
}

type dummyBSONStruct struct {
	A string
	B int
}

var dummyHintedNotBSONMarshalerHint = hint.MustHintWithType(hint.Type{0xff, 0x33}, "0.1", "dummyHintedNotBSONMarshaler")

type dummyHintedNotBSONMarshaler struct {
	A string
	B int
}

func (d dummyHintedNotBSONMarshaler) Hint() hint.Hint {
	return dummyHintedNotBSONMarshalerHint
}

var dummyHintedBSONMarshalerWithoutHintInfoHint = hint.MustHintWithType(hint.Type{0xff, 0x34}, "0.1", "dummyHintedBSONMarshalerWithoutHintInfo")

type dummyHintedBSONMarshalerWithoutHintInfo struct {
	A string
	B int
}

func (d dummyHintedBSONMarshalerWithoutHintInfo) Hint() hint.Hint {
	return dummyHintedBSONMarshalerWithoutHintInfoHint
}

func (d dummyHintedBSONMarshalerWithoutHintInfo) MarshalBSON() ([]byte, error) {
	return bson.Marshal(struct {
		A string
		B int
	}{A: d.A, B: d.B})
}

var dummyHintedBSONMarshalerWithHintInfoHint = hint.MustHintWithType(hint.Type{0xff, 0x35}, "0.1", "dummyHintedBSONMarshalerWithHintInfo")

type dummyHintedBSONMarshalerWithHintInfo struct {
	A string
	B int
}

func (d dummyHintedBSONMarshalerWithHintInfo) Hint() hint.Hint {
	return dummyHintedBSONMarshalerWithHintInfoHint
}

func (d dummyHintedBSONMarshalerWithHintInfo) MarshalBSON() ([]byte, error) {
	return bson.Marshal(struct {
		H hint.Hint `bson:"_hint"`
		A string
		B int
	}{H: d.Hint(), A: d.A, B: d.B})
}

type testBSON struct {
	suite.Suite
}

func (t *testBSON) TestEncodeNatives() {
	cases := []struct {
		name string
		v    interface{}
	}{
		/*
			{name: "nil", v: nil},
			{name: "string", v: util.UUID().String()},
			{name: "int", v: rand.Int()},
			{name: "int8", v: int8(33)},
			{name: "int16", v: int16(33)},
			{name: "int32", v: rand.Int31()},
			{name: "int64", v: rand.Int63()},
			{name: "uint", v: uint(rand.Int())},
			{name: "unt8", v: uint8(33)},
			{name: "unt16", v: uint16(33)},
			{name: "unt32", v: rand.Uint32()},
			{name: "unt64", v: rand.Uint64()},
			{name: "true", v: true},
			{name: "false", v: false},
			{name: "array", v: [3]int{3, 33, 333}},
			{name: "0 array", v: [3]int{}},
			{name: "array ptr", v: &([3]int{3, 33, 333})},
			{name: "slice", v: []int{3, 33, 333, 3333}},
			{name: "empty slice", v: []int{}},
			{name: "slice ptr", v: &([]int{3, 33, 333, 3333})},
		*/
		{
			name: "map",
			v:    map[string]int{util.UUID().String(): 1, util.UUID().String(): 2},
		},
		{
			name: "map ptr",
			v:    &map[string]int{util.UUID().String(): 1, util.UUID().String(): 2},
		},
		{name: "empty map", v: map[string]int{}},
		{name: "empty map ptr", v: &map[string]int{}},
	}

	je := NewBSONEncoder()

	for i, c := range cases {
		i := i
		c := c
		tested := t.Run(
			c.name,
			func() {
				b, err := je.Encode(c.v)
				t.NoError(err, "encode: %d: %v; error=%v", i, c.name, err)

				if c.v == nil {
					t.Nil(b, "%d: %v", i, c.name)
					return
				}

				n := reflect.New(reflect.TypeOf(c.v)).Interface()
				kind := reflect.TypeOf(c.v).Kind()
				if kind == reflect.Ptr {
					n = reflect.New(reflect.TypeOf(c.v).Elem()).Interface()
				}
				t.NoError(je.Decode(b, n), "decode: %d: %v", i, c.name)

				expected := c.v
				if kind == reflect.Ptr {
					expected = reflect.ValueOf(c.v).Elem().Interface()
				}
				t.Equal(expected, reflect.ValueOf(n).Elem().Interface(), "%d: %v", i, c.name)
			},
		)
		if !tested {
			break
		}
	}
}

func (t *testBSON) TestEncodeSimpleStruct() {
	s := s0{A: util.UUID().String()}

	je := NewBSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us s0
	t.NoError(je.Decode(b, &us))
	t.Equal(s.A, us.A)
}

func (t *testBSON) TestEncodeRLPPackable() {
	s := sp0{A: util.UUID().String()}

	je := NewBSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sp0
	t.NoError(je.Decode(b, &us))
	t.Equal(s.A, us.A)
	t.Nil(us.B)
}

func (t *testBSON) TestEncodeRLPUnpackable() {
	s := sup0{A: util.UUID().String()}

	je := NewBSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sup0
	t.NoError(je.Decode(b, &us))
	t.Equal(s.A+"-packed-unpacked", us.A)
}

func (t *testBSON) TestEncodeEmbed() {
	s := se0{
		A: util.UUID().String(),
		S: sup0{A: util.UUID().String()},
	}

	je := NewBSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us se0
	t.NoError(je.Decode(b, &us))

	t.Equal(s.A, us.A)
	t.Equal(s.S.A+"-packed", us.S.A)
}

func (t *testBSON) TestAnalyzePack() {
	je := NewBSONEncoder()

	{ // has PackRLP
		s := se0{
			A: util.UUID().String(),
			S: sup0{A: util.UUID().String()},
		}

		name, cp, err := je.analyze(s)
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal("default", name)
	}

	{ // don't have PackRLP
		s := s0{A: util.UUID().String()}

		name, cp, err := je.analyze(s)
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal(encoderAnalyzedTypeDefault, name)
	}

	{ // int-like
		name, cp, err := je.analyze(int(0))
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal(encoderAnalyzedTypeDefault, name)
	}

	{ // array
		name, cp, err := je.analyze([]int{1, 2})
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal(encoderAnalyzedTypeDefault, name)
	}

	{ // map
		name, cp, err := je.analyze(map[int]int{1: 1, 2: 2})
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal(encoderAnalyzedTypeDefault, name)
	}
}

func (t *testBSON) TestEncodeHinter() {
	s := sh0{B: util.UUID().String()}

	je := NewBSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sh0
	t.NoError(je.Decode(b, &us))

	t.Equal(s, us)
}

func (t *testBSON) TestEncodeHinterWithHead() {
	s := s1{C: rand.Int()}

	je := NewBSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us s1
	t.NoError(je.Decode(b, &us))

	t.Equal(s, us)
}

func (t *testBSON) TestEncodeHinterNotCompatible() {
	s := sh0{B: util.UUID().String()}

	je := NewBSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	{ // wrong major version
		c, _ := bson.Marshal(BSONPackHinted{
			H: hint.MustHint(sh0{}.Hint().Type(), "1.1"),
			D: s,
		})

		var us sh0
		err := je.Decode(c, &us)
		t.True(xerrors.Is(err, hint.VersionNotCompatibleError))
	}

	{ // wrong type version
		c, _ := bson.Marshal(BSONPackHinted{
			H: hint.MustHint(hint.Type{0xff, 0x30}, sh0{}.Hint().Version()),
			D: s,
		})

		var us sh0
		err := je.Decode(c, &us)
		t.Error(err)
	}
}

func (t *testBSON) TestAnonymousObject() {
	s := dummyBSONStruct{A: util.UUID().String(), B: 33}

	be := NewBSONEncoder()
	b, err := be.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var encoded []byte
	{
		b, err := be.Encode(s)
		t.NoError(err)

		encoded = b
	}

	var decoded dummyBSONStruct
	t.NoError(be.Decode(encoded, &decoded))

	t.Equal(s.A, decoded.A)
	t.Equal(s.B, decoded.B)
}

func (t *testBSON) TestHintedNotMarshaler() {
	encs := NewEncoders()
	be := NewBSONEncoder()
	t.NoError(encs.AddEncoder(be))
	t.NoError(encs.AddHinter(dummyHintedNotBSONMarshaler{}))

	s := dummyHintedNotBSONMarshaler{A: util.UUID().String(), B: 33}

	b, err := be.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var encoded []byte
	{
		b, err := be.Encode(s)
		t.NoError(err)

		encoded = b
	}

	hinter, err := be.DecodeByHint(encoded)
	t.NoError(err)
	t.IsType(dummyHintedNotBSONMarshaler{}, hinter)
}

func (t *testBSON) TestHintedMarshalerWithoutHint() {
	encs := NewEncoders()
	be := NewBSONEncoder()
	t.NoError(encs.AddEncoder(be))
	t.NoError(encs.AddHinter(dummyHintedBSONMarshalerWithoutHintInfo{}))

	s := dummyHintedBSONMarshalerWithoutHintInfo{A: util.UUID().String(), B: 33}

	encoded, err := be.Encode(s)
	t.NoError(err)
	t.NotNil(encoded)

	var decoded dummyHintedBSONMarshalerWithoutHintInfo
	t.NoError(be.Decode(encoded, &decoded))

	t.Equal(s.A, decoded.A)
	t.Equal(s.B, decoded.B)
}

func (t *testBSON) TestHintedMarshalerWithHint() {
	encs := NewEncoders()
	be := NewBSONEncoder()
	t.NoError(encs.AddEncoder(be))
	t.NoError(encs.AddHinter(dummyHintedBSONMarshalerWithHintInfo{}))

	s := dummyHintedBSONMarshalerWithHintInfo{A: util.UUID().String(), B: 33}

	encoded, err := be.Encode(s)
	t.NoError(err)
	t.NotNil(encoded)

	hinter, err := be.DecodeByHint(encoded)
	t.NoError(err)
	t.IsType(dummyHintedBSONMarshalerWithHintInfo{}, hinter)

	decoded := hinter.(dummyHintedBSONMarshalerWithHintInfo)

	t.Equal(s.A, decoded.A)
	t.Equal(s.B, decoded.B)
}

func (t *testBSON) TestMarshalWithJSON() {
	s := dummyBSONStruct{A: util.UUID().String(), B: 33}

	d := bson.M{}
	{
		b, err := bson.MarshalExtJSON(s, true, true)
		t.NoError(err)

		d["showme"] = util.UUID().String()
		d["json"] = b
	}

	b, err := bson.Marshal(d)
	t.NoError(err)

	var decoded bson.M
	t.NoError(bson.Unmarshal(b, &decoded))

	t.Equal(d["showme"], decoded["showme"])

	var ud dummyBSONStruct
	t.NoError(bson.UnmarshalExtJSON(decoded["json"].(primitive.Binary).Data, true, &ud))

	t.Equal(s, ud)
	t.Equal(s.A, ud.A)
	t.Equal(s.B, ud.B)
}

func TestBSON(t *testing.T) {
	suite.Run(t, new(testBSON))
}
