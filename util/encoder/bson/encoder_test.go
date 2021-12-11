package bsonenc

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
)

type hinterDefault struct {
	hint.BaseHinter
	A string
	B int
}

func newHinterDefault(ht hint.Hint, a string, b int) hinterDefault {
	return hinterDefault{
		BaseHinter: hint.NewBaseHinter(ht),
		A:          a,
		B:          b,
	}
}

func (ht hinterDefault) MarshalBSON() ([]byte, error) {
	return bson.Marshal(MergeBSONM(NewHintedDoc(ht.Hint()), bson.M{
		"A": ht.A,
		"B": ht.B,
	}))
}

type hinterBSONMarshaller struct {
	h hint.Hint
	a string
	b int
}

func (ht hinterBSONMarshaller) Hint() hint.Hint {
	return ht.h
}

func (ht hinterBSONMarshaller) SetHint(n hint.Hint) hint.Hinter {
	ht.h = n

	return ht
}

func (ht hinterBSONMarshaller) MarshalBSON() ([]byte, error) {
	return bson.Marshal(MergeBSONM(NewHintedDoc(ht.h), bson.M{
		"A": ht.a,
		"B": ht.b,
	}))
}

func (ht *hinterBSONMarshaller) UnmarshalBSON(b []byte) error {
	var uht struct {
		A string
		B int
	}
	if err := bson.Unmarshal(b, &uht); err != nil {
		return err
	}

	ht.a = uht.A
	ht.b = uht.B

	return nil
}

type hinterBSONUnpacker struct {
	hint.BaseHinter
	a string
	b int
}

func newHinterBSONUnpacker(ht hint.Hint, a string, b int) hinterBSONUnpacker {
	return hinterBSONUnpacker{
		BaseHinter: hint.NewBaseHinter(ht),
		a:          a,
		b:          b,
	}
}

func (ht hinterBSONUnpacker) MarshalBSON() ([]byte, error) {
	return bson.Marshal(MergeBSONM(NewHintedDoc(ht.Hint()), bson.M{
		"A": ht.a, "B": ht.b,
	}))
}

func (ht *hinterBSONUnpacker) UnpackBSON(b []byte, enc *Encoder) error {
	var uht struct {
		A string
		B int
	}
	if err := enc.Unmarshal(b, &uht); err != nil {
		return err
	}

	ht.a = uht.A
	ht.b = uht.B

	return nil
}

type hinterSlice struct {
	H  hint.Hint
	HS []hint.Hinter
}

func (ht hinterSlice) Hint() hint.Hint {
	return ht.H
}

func (ht hinterSlice) MarshalBSON() ([]byte, error) {
	return bson.Marshal(MergeBSONM(NewHintedDoc(ht.H), bson.M{
		"HS": ht.HS,
	}))
}

func (ht *hinterSlice) UnpackBSON(b []byte, enc *Encoder) error {
	var u struct {
		HS bson.Raw
	}

	if err := bson.Unmarshal(b, &u); err != nil {
		return err
	}

	i, err := enc.DecodeSlice(u.HS)
	if err != nil {
		return err
	}

	ht.HS = i

	return nil
}

type testBSONEncoder struct {
	suite.Suite
}

func (t *testBSONEncoder) TestNew() {
	enc := NewEncoder()
	_, ok := (interface{})(enc).(encoder.Encoder)
	t.True(ok)
}

func (t *testBSONEncoder) TestAdd() {
	enc := NewEncoder()

	ht := newHinterDefault(
		hint.NewHint(hint.Type("findme"), "v1.2.3"),
		"A", 33,
	)
	t.NoError(enc.Add(ht))

	// add again
	err := enc.Add(ht)
	t.Contains(err.Error(), "already added")
}

func (t *testBSONEncoder) TestDecodeUnknown() {
	enc := NewEncoder()

	ht := newHinterDefault(
		hint.NewHint(hint.Type("findme"), "v1.2.3"),
		"A", 33,
	)

	b, err := enc.Marshal(ht)
	t.NoError(err)

	_, err = enc.Decode(b)
	t.True(errors.Is(err, util.NotFoundError))
}

func (t *testBSONEncoder) TestDecodeDefault() {
	enc := NewEncoder()

	ht := newHinterDefault(
		hint.NewHint(hint.Type("findme"), "v1.2.3"),
		"A",
		33,
	)
	t.NoError(enc.Add(ht))

	b, err := enc.Marshal(ht)
	t.NoError(err)

	hinter, err := enc.Decode(b)
	t.NoError(err)

	uht, ok := hinter.(hinterDefault)
	t.True(ok)

	t.Equal(ht.A, uht.A)
	t.Equal(ht.B, uht.B)
}

func (t *testBSONEncoder) TestDecodeBSONUnmarshaller() {
	enc := NewEncoder()

	ht := hinterBSONMarshaller{
		h: hint.NewHint(hint.Type("findme"), "v1.2.3"),
	}
	t.NoError(enc.Add(ht))

	ht.a = "fa"
	ht.b = 33

	b, err := enc.Marshal(ht)
	t.NoError(err)

	hinter, err := enc.Decode(b)
	t.NoError(err)

	uht, ok := hinter.(hinterBSONMarshaller)
	t.True(ok)

	t.True(ht.Hint().Equal(uht.Hint()))
	t.Equal(ht.a, uht.a)
	t.Equal(ht.b, uht.b)
}

func (t *testBSONEncoder) TestDecodeBSONUnpacker() {
	enc := NewEncoder()

	ht := newHinterBSONUnpacker(
		hint.NewHint(hint.Type("findme"), "v1.2.3"),
		"fa",
		33,
	)
	t.NoError(enc.Add(ht))

	b, err := enc.Marshal(ht)
	t.NoError(err)

	hinter, err := enc.Decode(b)
	t.NoError(err)

	uht, ok := hinter.(hinterBSONUnpacker)
	t.True(ok)

	t.Equal(ht.a, uht.a)
	t.Equal(ht.b, uht.b)
}

func (t *testBSONEncoder) TestDecodeSlice() {
	enc := NewEncoder()

	htj := hinterBSONMarshaller{
		h: hint.NewHint(hint.Type("text"), "v1.2.3"),
	}
	htu := newHinterBSONUnpacker(
		hint.NewHint(hint.Type("unpack"), "v1.2.3"),
		"",
		0,
	)

	hs := hinterSlice{
		H: hint.NewHint(hint.Type("slice"), "v1.2.3"),
	}
	t.NoError(enc.Add(htj))
	t.NoError(enc.Add(htu))
	t.NoError(enc.Add(hs))

	ht0 := hinterBSONMarshaller{h: htj.Hint(), a: "A", b: 44}
	ht1 := newHinterBSONUnpacker(htu.Hint(), "a", 33)

	hs.HS = []hint.Hinter{ht0, ht1}

	b, err := enc.Marshal(hs)
	t.NoError(err)

	i, err := enc.Decode(b)
	t.NoError(err)

	uhs, ok := i.(hinterSlice)
	t.True(ok)

	t.Equal(2, len(uhs.HS))

	uht0, ok := uhs.HS[0].(hinterBSONMarshaller)
	t.True(ok)
	uht1, ok := uhs.HS[1].(hinterBSONUnpacker)
	t.True(ok)

	t.Equal(ht0.a, uht0.a)
	t.Equal(ht0.b, uht0.b)

	t.Equal(ht1.a, uht1.a)
	t.Equal(ht1.b, uht1.b)
}

func (t *testBSONEncoder) TestDecodeMap() {
	enc := NewEncoder()

	htj := hinterBSONMarshaller{
		h: hint.NewHint(hint.Type("text"), "v1.2.3"),
	}
	htu := newHinterBSONUnpacker(
		hint.NewHint(hint.Type("unpack"), "v1.2.3"),
		"",
		0,
	)
	t.NoError(enc.Add(htj))
	t.NoError(enc.Add(htu))

	ht0 := hinterBSONMarshaller{h: htj.Hint(), a: "A", b: 44}
	ht1 := newHinterBSONUnpacker(htu.Hint(), "a", 33)

	b, err := enc.Marshal(map[string]interface{}{
		"ht0": ht0,
		"ht1": ht1,
	})
	t.NoError(err)

	i, err := enc.DecodeMap(b)
	t.NoError(err)

	t.Equal(2, len(i))

	uht0, ok := i["ht0"].(hinterBSONMarshaller)
	t.True(ok)
	uht1, ok := i["ht1"].(hinterBSONUnpacker)
	t.True(ok)

	t.Equal(ht0.a, uht0.a)
	t.Equal(ht0.b, uht0.b)

	t.Equal(ht1.a, uht1.a)
	t.Equal(ht1.b, uht1.b)
}

func (t *testBSONEncoder) TestDecodeKeepHint() {
	enc := NewEncoder()

	orig := newHinterDefault(
		hint.NewHint(hint.Type("findme"), "v1.2.3"),
		"",
		0,
	)
	t.NoError(enc.Add(orig))

	ht := newHinterDefault(
		hint.NewHint(hint.Type("findme"), "v1.2.0"),
		"A",
		33,
	)

	t.Equal(orig.Hint().Type(), ht.Hint().Type())
	t.NotEqual(orig.Hint().Version(), ht.Hint().Version())

	b, err := enc.Marshal(ht)
	t.NoError(err)

	hinter, err := enc.Decode(b)
	t.NoError(err)

	uht, ok := hinter.(hinterDefault)
	t.True(ok)

	t.True(ht.Hint().Equal(uht.Hint()))

	t.Equal(ht.A, uht.A)
	t.Equal(ht.B, uht.B)
}

func TestBSONEncoder(t *testing.T) {
	suite.Run(t, new(testBSONEncoder))
}
