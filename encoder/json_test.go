package encoder

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/util"
)

var s1Hint = hint.MustHintWithType(hint.Type{0xff, 0x32}, "0.1", "s1")

func (s0 sp0) PackJSON(_ *JSONEncoder) (interface{}, error) {
	return sp0{
		A: s0.A,
		B: []byte(s0.A),
	}, nil
}

func (s0 sup0) PackJSON(_ *JSONEncoder) (interface{}, error) {
	return sup0{
		A: s0.A + "-packed",
	}, nil
}

func (s0 *sup0) UnpackJSON(b []byte, _ *JSONEncoder) error {
	var us sup0
	if err := json.Unmarshal(b, &us); err != nil {
		return err
	}

	s0.A = us.A + "-unpacked"

	return nil
}

// NOTE embed struct must have PackJSON and UnpackJSON.
func (s0 se0) PackJSON(rp *JSONEncoder) (interface{}, error) {
	s, err := rp.Pack(s0.S)
	if err != nil {
		return nil, err
	}

	return struct {
		A string
		S interface{}
	}{
		A: s0.A,
		S: s,
	}, nil
}

func (s0 *se0) UnpackJSON(b []byte, rp *JSONEncoder) error {
	var se struct {
		A string
		S json.RawMessage
	}
	if err := json.Unmarshal(b, &se); err != nil {
		return err
	}

	var sup sup0
	if err := rp.Unpack(se.S, &sup); err != nil {
		return err
	}

	s0.A = se.A
	s0.S = sup

	return nil
}

func (s0 sh0) PackJSON(_ *JSONEncoder) (interface{}, error) {
	return struct {
		JSONPackHintedHead
		sh0
	}{
		JSONPackHintedHead: NewJSONPackHintedHead(s0.Hint()),
		sh0:                s0,
	}, nil
}

type s1 struct {
	C int
}

// s1 does not PackJSON without JSONPackHintedHead
func (s0 s1) Hint() hint.Hint {
	return s1Hint
}

type testJSON struct {
	suite.Suite
}

func (t *testJSON) TestEncodeNatives() {
	cases := []struct {
		name string
		v    interface{}
	}{
		//{name: "nil", v: nil},
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

	je := NewJSONEncoder()

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

func (t *testJSON) TestEncodeSimpleStruct() {
	s := s0{A: util.UUID().String()}

	je := NewJSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us s0
	t.NoError(je.Decode(b, &us))
	t.Equal(s.A, us.A)
}

func (t *testJSON) TestEncodeRLPPackable() {
	s := sp0{A: util.UUID().String()}

	je := NewJSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sp0
	t.NoError(je.Decode(b, &us))
	t.Equal(s.A, us.A)
	t.Equal([]byte(s.A), us.B)
}

func (t *testJSON) TestEncodeRLPUnpackable() {
	s := sup0{A: util.UUID().String()}

	je := NewJSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sup0
	t.NoError(je.Decode(b, &us))
	t.Equal(s.A+"-packed-unpacked", us.A)
}

func (t *testJSON) TestEncodeEmbed() {
	s := se0{
		A: util.UUID().String(),
		S: sup0{A: util.UUID().String()},
	}

	je := NewJSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us se0
	t.NoError(je.Decode(b, &us))

	t.Equal(s.A, us.A)
	t.Equal(s.S.A+"-packed-unpacked", us.S.A)
}

func (t *testJSON) TestAnalyzePack() {
	je := NewJSONEncoder()

	{ // has PackRLP
		s := se0{
			A: util.UUID().String(),
			S: sup0{A: util.UUID().String()},
		}

		name, cp, err := je.analyze(s)
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal([]string{"JSONPackable", "JSONUnpackable"}, name)
	}

	{ // don't have PackRLP
		s := s0{A: util.UUID().String()}

		name, cp, err := je.analyze(s)
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal([]string{encoderAnalyzedTypeDefault, encoderAnalyzedTypeDefault}, name)
	}

	{ // int-like
		name, cp, err := je.analyze(int(0))
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal([]string{encoderAnalyzedTypeDefault, encoderAnalyzedTypeDefault}, name)
	}

	{ // array
		name, cp, err := je.analyze([]int{1, 2})
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal([]string{encoderAnalyzedTypeDefault, encoderAnalyzedTypeDefault}, name)
	}

	{ // map
		name, cp, err := je.analyze(map[int]int{1: 1, 2: 2})
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal([]string{encoderAnalyzedTypeDefault, encoderAnalyzedTypeDefault}, name)
	}
}

func (t *testJSON) TestEncodeHinter() {
	s := sh0{B: util.UUID().String()}

	je := NewJSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sh0
	t.NoError(je.Decode(b, &us))

	t.Equal(s, us)
}

func (t *testJSON) TestEncodeHinterWithHead() {
	s := s1{C: rand.Int()}

	je := NewJSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us s1
	t.NoError(je.Decode(b, &us))

	t.Equal(s, us)
}

func (t *testJSON) TestEncodeHinterNotCompatible() {
	s := sh0{B: util.UUID().String()}

	je := NewJSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	{ // wrong major version
		c := bytes.Replace(b, []byte(`"version":"0.1"`), []byte(`"version":"1.1"`), -1)

		var us sh0
		err := je.Decode(c, &us)
		t.True(xerrors.Is(err, hint.VersionNotCompatibleError))
	}

	{ // wrong type version
		c := bytes.Replace(b, []byte(`"type":{"name":"sh0"`), []byte(`"type":{"name":"sh1"`), -1)

		var us sh0
		err := je.Decode(c, &us)
		t.Error(err)
	}
}

func TestJSON(t *testing.T) {
	suite.Run(t, new(testJSON))
}
