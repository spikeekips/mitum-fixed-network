package bsonenc

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

// s0 is simple struct
type s0 struct {
	A string
}

// sup0 has UnpackRLP
type sup0 struct {
	A string
}

func (s0 sup0) MarshalBSON() ([]byte, error) {
	return Marshal(struct {
		A string
	}{
		A: s0.A + "-packed",
	})
}

func (s0 *sup0) UnpackBSON(b []byte, _ *Encoder) error {
	var us sup0
	if err := Unmarshal(b, &us); err != nil {
		return err
	}

	s0.A = us.A + "-unpacked"

	return nil
}

// sp0 has PackRLP
type sp0 struct {
	A string
	B []byte
}

// se0 embeds sup0
type se0 struct {
	A string
	S sup0
}

var sh0Hint = hint.MustHintWithType(hint.Type{0xff, 0x31}, "0.1", "sh0")

type sh0 struct {
	B string
}

func (s0 sh0) Hint() hint.Hint {
	return sh0Hint
}

var s1Hint = hint.MustHintWithType(hint.Type{0xff, 0x32}, "0.1", "s1")

type s1 struct {
	C int
}

func (s0 s1) Hint() hint.Hint {
	return s1Hint
}

type dummyStruct struct {
	A string
	B int
}

var dummyHintedNotMarshalerHint = hint.MustHintWithType(hint.Type{0xff, 0x33}, "0.1", "dummyHintedNotMarshaler")

type dummyHintedNotMarshaler struct {
	A string
	B int
}

func (d dummyHintedNotMarshaler) Hint() hint.Hint {
	return dummyHintedNotMarshalerHint
}

var dummyHintedMarshalerWithoutHintInfoHint = hint.MustHintWithType(hint.Type{0xff, 0x34}, "0.1", "dummyHintedMarshalerWithoutHintInfo")

type dummyHintedMarshalerWithoutHintInfo struct {
	A string
	B int
}

func (d dummyHintedMarshalerWithoutHintInfo) Hint() hint.Hint {
	return dummyHintedMarshalerWithoutHintInfoHint
}

func (d dummyHintedMarshalerWithoutHintInfo) MarshalBSON() ([]byte, error) {
	return Marshal(struct {
		A string
		B int
	}{A: d.A, B: d.B})
}

var dummyHintedMarshalerWithHintInfoHint = hint.MustHintWithType(hint.Type{0xff, 0x35}, "0.1", "dummyHintedMarshalerWithHintInfo")

type dummyHintedMarshalerWithHintInfo struct {
	A string
	B int
}

func (d dummyHintedMarshalerWithHintInfo) Hint() hint.Hint {
	return dummyHintedMarshalerWithHintInfoHint
}

func (d dummyHintedMarshalerWithHintInfo) MarshalBSON() ([]byte, error) {
	return Marshal(struct {
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

	be := NewEncoder()

	for i, c := range cases {
		i := i
		c := c
		tested := t.Run(
			c.name,
			func() {
				b, err := be.Marshal(c.v)
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
				t.NoError(be.Decode(b, n), "decode: %d: %v", i, c.name)

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

	be := NewEncoder()
	b, err := be.Marshal(s)
	t.NoError(err)
	t.NotNil(b)

	var us s0
	t.NoError(be.Decode(b, &us))
	t.Equal(s.A, us.A)
}

func (t *testBSON) TestEncodeEmbed() {
	s := se0{
		A: util.UUID().String(),
		S: sup0{A: util.UUID().String()},
	}

	be := NewEncoder()
	b, err := be.Marshal(s)
	t.NoError(err)
	t.NotNil(b)

	var us se0
	t.NoError(be.Decode(b, &us))

	t.Equal(s.A, us.A)
	t.Equal(s.S.A+"-packed", us.S.A)
}

func (t *testBSON) TestAnalyzePack() {
	be := NewEncoder()

	{ // has PackRLP
		s := se0{
			A: util.UUID().String(),
			S: sup0{A: util.UUID().String()},
		}

		name, cp, err := be.analyze(s)
		t.NoError(err)
		t.NotNil(cp.Unpack)
		t.Equal("default", name)
	}

	{ // don't have PackRLP
		s := s0{A: util.UUID().String()}

		name, cp, err := be.analyze(s)
		t.NoError(err)
		t.NotNil(cp.Unpack)
		t.Equal(encoder.EncoderAnalyzedTypeDefault, name)
	}

	{ // int-like
		name, cp, err := be.analyze(int(0))
		t.NoError(err)
		t.NotNil(cp.Unpack)
		t.Equal(encoder.EncoderAnalyzedTypeDefault, name)
	}

	{ // array
		name, cp, err := be.analyze([]int{1, 2})
		t.NoError(err)
		t.NotNil(cp.Unpack)
		t.Equal(encoder.EncoderAnalyzedTypeDefault, name)
	}

	{ // map
		name, cp, err := be.analyze(map[int]int{1: 1, 2: 2})
		t.NoError(err)
		t.NotNil(cp.Unpack)
		t.Equal(encoder.EncoderAnalyzedTypeDefault, name)
	}
}

func (t *testBSON) TestEncodeHinter() {
	s := sh0{B: util.UUID().String()}

	be := NewEncoder()
	b, err := be.Marshal(s)
	t.NoError(err)
	t.NotNil(b)

	var us sh0
	t.NoError(be.Decode(b, &us))

	t.Equal(s, us)
}

func (t *testBSON) TestEncodeHinterWithHead() {
	s := s1{C: rand.Int()}

	be := NewEncoder()
	b, err := be.Marshal(s)
	t.NoError(err)
	t.NotNil(b)

	var us s1
	t.NoError(be.Decode(b, &us))

	t.Equal(s, us)
}

func (t *testBSON) TestEncodeHinterNotCompatible() {
	s := struct {
		H hint.Hint `bson:"_hint"`
		B string
	}{
		H: (sh0{}).Hint(),
		B: util.UUID().String(),
	}

	encs := encoder.NewEncoders()
	be := NewEncoder()
	encs.AddEncoder(be)
	encs.AddHinter(sh0{})

	var encoded []byte
	{
		b, err := be.Marshal(s)
		t.NoError(err)
		t.NotNil(b)

		var m0 bson.M
		t.NoError(Unmarshal(b, &m0))
		nb, err := jsonenc.Marshal(m0)
		t.NoError(err)

		encoded = nb
	}

	{ // wrong major version
		var decoded []byte
		{
			c := bytes.Replace(encoded, []byte(`+0.1`), []byte(`+1.1`), -1)

			var m1 bson.M
			t.NoError(jsonenc.Unmarshal(c, &m1))

			d, err := Marshal(m1)
			t.NoError(err)

			decoded = d
		}

		_, err := be.DecodeByHint(decoded)
		t.True(xerrors.Is(err, hint.HintNotFoundError))
	}

	{ // wrong type code
		var decoded []byte
		{
			c := bytes.Replace(encoded, []byte(`ff31+`), []byte(`ffaa+`), -1)

			var m1 bson.M
			t.NoError(jsonenc.Unmarshal(c, &m1))

			d, err := Marshal(m1)
			t.NoError(err)

			decoded = d
		}

		_, err := be.DecodeByHint(decoded)
		t.Contains(err.Error(), "Hint not found in Hintset")
	}
}

func (t *testBSON) TestAnonymousObject() {
	s := dummyStruct{A: util.UUID().String(), B: 33}

	be := NewEncoder()
	b, err := be.Marshal(s)
	t.NoError(err)
	t.NotNil(b)

	var encoded []byte
	{
		b, err := be.Marshal(s)
		t.NoError(err)

		encoded = b
	}

	var decoded dummyStruct
	t.NoError(be.Decode(encoded, &decoded))

	t.Equal(s.A, decoded.A)
	t.Equal(s.B, decoded.B)
}

func (t *testBSON) TestHintedNotMarshaler() {
	encs := encoder.NewEncoders()
	be := NewEncoder()
	t.NoError(encs.AddEncoder(be))
	t.NoError(encs.AddHinter(dummyHintedNotMarshaler{}))

	s := dummyHintedNotMarshaler{
		A: util.UUID().String(),
		B: 33,
	}

	b, err := be.Marshal(MergeBSONM(
		NewHintedDoc(s.Hint()),
		bson.M{
			"A": s.A,
			"B": s.B,
		},
	))
	t.NoError(err)
	t.NotNil(b)

	hinter, err := be.DecodeByHint(b)
	t.NoError(err)
	t.IsType(dummyHintedNotMarshaler{}, hinter)
}

func (t *testBSON) TestHintedMarshalerWithoutHint() {
	encs := encoder.NewEncoders()
	be := NewEncoder()
	t.NoError(encs.AddEncoder(be))
	t.NoError(encs.AddHinter(dummyHintedMarshalerWithoutHintInfo{}))

	s := dummyHintedMarshalerWithoutHintInfo{A: util.UUID().String(), B: 33}

	encoded, err := be.Marshal(s)
	t.NoError(err)
	t.NotNil(encoded)

	var decoded dummyHintedMarshalerWithoutHintInfo
	t.NoError(be.Decode(encoded, &decoded))

	t.Equal(s.A, decoded.A)
	t.Equal(s.B, decoded.B)
}

func (t *testBSON) TestHintedMarshalerWithHint() {
	encs := encoder.NewEncoders()
	be := NewEncoder()
	t.NoError(encs.AddEncoder(be))
	t.NoError(encs.AddHinter(dummyHintedMarshalerWithHintInfo{}))

	s := dummyHintedMarshalerWithHintInfo{A: util.UUID().String(), B: 33}

	encoded, err := be.Marshal(s)
	t.NoError(err)
	t.NotNil(encoded)

	hinter, err := be.DecodeByHint(encoded)
	t.NoError(err)
	t.IsType(dummyHintedMarshalerWithHintInfo{}, hinter)

	decoded := hinter.(dummyHintedMarshalerWithHintInfo)

	t.Equal(s.A, decoded.A)
	t.Equal(s.B, decoded.B)
}

func (t *testBSON) TestMarshalWithJSON() {
	s := dummyStruct{A: util.UUID().String(), B: 33}

	d := bson.M{}
	{
		b, err := bson.MarshalExtJSON(s, true, true)
		t.NoError(err)

		d["showme"] = util.UUID().String()
		d["json"] = b
	}

	b, err := Marshal(d)
	t.NoError(err)

	var decoded bson.M
	t.NoError(Unmarshal(b, &decoded))

	t.Equal(d["showme"], decoded["showme"])

	var ud dummyStruct
	t.NoError(bson.UnmarshalExtJSON(decoded["json"].(primitive.Binary).Data, true, &ud))

	t.Equal(s, ud)
	t.Equal(s.A, ud.A)
	t.Equal(s.B, ud.B)
}

func TestBSON(t *testing.T) {
	suite.Run(t, new(testBSON))
}
