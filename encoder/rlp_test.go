package encoder

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/util"
)

// s0 is simple struct
type s0 struct {
	A string
}

// sp0 has PackRLP
type sp0 struct {
	A string
	B []byte
}

func (s0 sp0) PackRLP(_ *RLPEncoder) (interface{}, error) {
	return sp0{
		A: s0.A,
		B: []byte(s0.A),
	}, nil
}

// sup0 has UnpackRLP
type sup0 struct {
	A string
}

func (s0 sup0) PackRLP(_ *RLPEncoder) (interface{}, error) {
	return sup0{
		A: s0.A + "-packed",
	}, nil
}

func (s0 *sup0) UnpackRLP(stream *rlp.Stream, _ *RLPEncoder) error {
	var us sup0
	if err := stream.Decode(&us); err != nil {
		return err
	}

	s0.A = us.A + "-unpacked"

	return nil
}

// se0 embeds sup0
type se0 struct {
	A string
	S sup0
}

// NOTE embed struct must have PackRLP and UnpackRLP.
func (s0 se0) PackRLP(rp *RLPEncoder) (interface{}, error) {
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

func (s0 *se0) UnpackRLP(stream *rlp.Stream, rp *RLPEncoder) error {
	var se struct {
		A string
		S rlp.RawValue
	}
	if err := stream.Decode(&se); err != nil {
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

type sh0 struct {
	B string
}

func (s0 sh0) Hint() hint.Hint {
	return hint.MustHint(hint.Type{0xff, 0x31}, "0.1")
}

type testRLP struct {
	suite.Suite
}

func (t *testRLP) TestEncodeNatives() {
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
		{name: "map",
			v: map[string]int{util.UUID().String(): 1, util.UUID().String(): 2}},
		{name: "map ptr",
			v: &map[string]int{util.UUID().String(): 1, util.UUID().String(): 2}},
		{name: "empty map", v: map[string]int{}},
		{name: "empty map ptr", v: &map[string]int{}},
	}

	re := NewRLPEncoder()

	for i, c := range cases {
		i := i
		c := c
		tested := t.Run(
			c.name,
			func() {
				b, err := re.Encode(c.v)
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
				t.NoError(re.Decode(b, n), "decode: %d: %v", i, c.name)

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

func (t *testRLP) TestEncodeSimpleStruct() {
	s := s0{A: util.UUID().String()}

	re := NewRLPEncoder()
	b, err := re.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us s0
	t.NoError(re.Decode(b, &us))
	t.Equal(s.A, us.A)
}

func (t *testRLP) TestEncodeRLPPackable() {
	s := sp0{A: util.UUID().String()}

	re := NewRLPEncoder()
	b, err := re.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sp0
	t.NoError(re.Decode(b, &us))
	t.Equal(s.A, us.A)
	t.Equal([]byte(s.A), us.B)
}

func (t *testRLP) TestEncodeRLPUnpackable() {
	s := sup0{A: util.UUID().String()}

	re := NewRLPEncoder()
	b, err := re.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sup0
	t.NoError(re.Decode(b, &us))
	t.Equal(s.A+"-packed-unpacked", us.A)
}

func (t *testRLP) TestEncodeEmbed() {
	s := se0{
		A: util.UUID().String(),
		S: sup0{A: util.UUID().String()},
	}

	re := NewRLPEncoder()
	b, err := re.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us se0
	t.NoError(re.Decode(b, &us))

	t.Equal(s.A, us.A)
	t.Equal(s.S.A+"-packed-unpacked", us.S.A)
}

func (t *testRLP) TestAnalyzePack() {
	re := NewRLPEncoder()

	{ // has PackRLP
		s := se0{
			A: util.UUID().String(),
			S: sup0{A: util.UUID().String()},
		}

		name, cp, err := re.analyze(s)
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal("RLPPackable+RLPUnpackable", name)
	}

	{ // don't have PackRLP
		s := s0{A: util.UUID().String()}

		name, cp, err := re.analyze(s)
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal(encoderAnalyzedTypeDefault, name)
	}

	{ // int-like
		name, cp, err := re.analyze(int(0))
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal("packValueInt", name)
	}

	{ // array
		name, cp, err := re.analyze([]int{1, 2})
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal("packValueArray", name)
	}

	{ // map
		name, cp, err := re.analyze(map[int]int{1: 1, 2: 2})
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal("packValueMap", name)
	}
}

func (t *testRLP) TestEncodeHinter() {
	s := sh0{B: util.UUID().String()}

	re := NewRLPEncoder()
	b, err := re.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sh0
	t.NoError(re.Decode(b, &us))

	t.Equal(s, us)
}

func TestRLP(t *testing.T) {
	suite.Run(t, new(testRLP))
}
