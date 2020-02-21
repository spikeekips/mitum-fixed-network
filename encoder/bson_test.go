package encoder

import (
	"math/rand"
	"reflect"
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
)

func (s0 sp0) PackBSON(_ *BSONEncoder) (interface{}, error) {
	return sp0{
		A: s0.A,
		B: []byte(s0.A),
	}, nil
}

func (s0 sup0) PackBSON(_ *BSONEncoder) (interface{}, error) {
	return sup0{
		A: s0.A + "-packed",
	}, nil
}

func (s0 *sup0) UnpackBSON(b []byte, _ *BSONEncoder) error {
	var us sup0
	if err := bson.Unmarshal(b, &us); err != nil {
		return err
	}

	s0.A = us.A + "-unpacked"

	return nil
}

// NOTE embed struct must have PackBSON and UnpackBSON.
func (s0 se0) PackBSON(rp *BSONEncoder) (interface{}, error) {
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

func (s0 *se0) UnpackBSON(b []byte, rp *BSONEncoder) error {
	var se struct {
		A string
		S bson.Raw
	}
	if err := bson.Unmarshal(b, &se); err != nil {
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

func (s0 sh0) PackBSON(_ *BSONEncoder) (interface{}, error) {
	return s0, nil
}

type testBSON struct {
	suite.Suite
}

func (t *testBSON) TestEncodeNatives() {
	cases := []struct {
		name string
		v    interface{}
	}{
		//{name: "nil", v: nil},
		/*
			{name: "string", v: uuid.Must(uuid.NewV4(), nil).String()},
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
		{name: "map",
			v: map[string]int{uuid.Must(uuid.NewV4(), nil).String(): 1, uuid.Must(uuid.NewV4(), nil).String(): 2}},
		{name: "map ptr",
			v: &map[string]int{uuid.Must(uuid.NewV4(), nil).String(): 1, uuid.Must(uuid.NewV4(), nil).String(): 2}},
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
				return

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
	s := s0{A: uuid.Must(uuid.NewV4(), nil).String()}

	je := NewBSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us s0
	t.NoError(je.Decode(b, &us))
	t.Equal(s.A, us.A)
}

func (t *testBSON) TestEncodeRLPPackable() {
	s := sp0{A: uuid.Must(uuid.NewV4(), nil).String()}

	je := NewBSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sp0
	t.NoError(je.Decode(b, &us))
	t.Equal(s.A, us.A)
	t.Equal([]byte(s.A), us.B)
}

func (t *testBSON) TestEncodeRLPUnpackable() {
	s := sup0{A: uuid.Must(uuid.NewV4(), nil).String()}

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
		A: uuid.Must(uuid.NewV4(), nil).String(),
		S: sup0{A: uuid.Must(uuid.NewV4(), nil).String()},
	}

	je := NewBSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us se0
	t.NoError(je.Decode(b, &us))

	t.Equal(s.A, us.A)
	t.Equal(s.S.A+"-packed-unpacked", us.S.A)
}

func (t *testBSON) TestAnalyzePack() {
	je := NewBSONEncoder()

	{ // has PackRLP
		s := se0{
			A: uuid.Must(uuid.NewV4(), nil).String(),
			S: sup0{A: uuid.Must(uuid.NewV4(), nil).String()},
		}

		name, cp, err := je.analyze(s)
		t.NoError(err)
		t.NotNil(cp.Pack)
		t.NotNil(cp.Unpack)
		t.Equal("BSONPackable+BSONUnpackable", name)
	}

	{ // don't have PackRLP
		s := s0{A: uuid.Must(uuid.NewV4(), nil).String()}

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
	_ = hint.RegisterType(sh0{}.Hint().Type(), "sh0")

	s := sh0{B: uuid.Must(uuid.NewV4(), nil).String()}

	je := NewBSONEncoder()
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sh0
	t.NoError(je.Decode(b, &us))

	t.Equal(s, us)
}

func (t *testBSON) TestEncodeHinterWithHead() {
	_ = hint.RegisterType(s1{}.Hint().Type(), "sh1")

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
	_ = hint.RegisterType(sh0{}.Hint().Type(), "sh0")

	s := sh0{B: uuid.Must(uuid.NewV4(), nil).String()}

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

func TestBSON(t *testing.T) {
	suite.Run(t, new(testBSON))
}
