package jsonenc

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
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

func (ht hinterDefault) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(struct {
		HintedHead
		A string
		B int
	}{
		HintedHead: NewHintedHead(ht.Hint()),
		A:          ht.A, B: ht.B,
	})
}

type hinterJSONMarshaller struct {
	h hint.Hint
	a string
	b int
}

func (ht hinterJSONMarshaller) Hint() hint.Hint {
	return ht.h
}

func (ht hinterJSONMarshaller) SetHint(n hint.Hint) hint.Hinter {
	ht.h = n

	return ht
}

func (ht hinterJSONMarshaller) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(struct {
		HintedHead
		A string
		B int
	}{
		HintedHead: NewHintedHead(ht.Hint()),
		A:          ht.a, B: ht.b,
	})
}

func (ht *hinterJSONMarshaller) UnmarshalJSON(b []byte) error {
	var uht struct {
		A string
		B int
	}
	if err := util.JSON.Unmarshal(b, &uht); err != nil {
		return err
	}

	ht.a = uht.A
	ht.b = uht.B

	return nil
}

type hinterJSONUnpacker struct {
	hint.BaseHinter
	a string
	b int
}

func newHinterJSONUnpacker(ht hint.Hint, a string, b int) hinterJSONUnpacker {
	return hinterJSONUnpacker{
		BaseHinter: hint.NewBaseHinter(ht),
		a:          a,
		b:          b,
	}
}

func (ht hinterJSONUnpacker) MarshalJSON() ([]byte, error) {
	return util.JSON.Marshal(struct {
		HintedHead
		A string
		B int
	}{
		HintedHead: NewHintedHead(ht.Hint()),
		A:          ht.a, B: ht.b,
	})
}

func (ht *hinterJSONUnpacker) UnpackJSON(b []byte, enc *Encoder) error {
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

type testJSONEncoder struct {
	suite.Suite
}

func (t *testJSONEncoder) TestNew() {
	enc := NewEncoder()
	_, ok := (interface{})(enc).(encoder.Encoder)
	t.True(ok)
}

func (t *testJSONEncoder) TestAdd() {
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

func (t *testJSONEncoder) TestDecodeUnknown() {
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

func (t *testJSONEncoder) TestDecodeDefault() {
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

func (t *testJSONEncoder) TestDecodeJSONUnmarshaller() {
	enc := NewEncoder()

	orig := hinterJSONMarshaller{
		h: hint.NewHint(hint.Type("findme"), "v1.2.3"),
	}
	t.NoError(enc.Add(orig))

	ht := hinterJSONMarshaller{
		h: hint.NewHint(hint.Type("findme"), "v1.2.1"),
		a: "fa",
		b: 33,
	}

	b, err := enc.Marshal(ht)
	t.NoError(err)

	hinter, err := enc.Decode(b)
	t.NoError(err)

	uht, ok := hinter.(hinterJSONMarshaller)
	t.True(ok)

	t.True(ht.Hint().Equal(uht.Hint()))
	t.Equal(ht.a, uht.a)
	t.Equal(ht.b, uht.b)
}

func (t *testJSONEncoder) TestDecodeJSONUnpacker() {
	enc := NewEncoder()

	orig := newHinterJSONUnpacker(
		hint.NewHint(hint.Type("findme"), "v1.2.3"),
		"",
		0,
	)
	t.NoError(enc.Add(orig))

	ht := newHinterJSONUnpacker(
		hint.NewHint(hint.Type("findme"), "v1.2.2"),
		"fa",
		33,
	)
	t.Equal(orig.Hint().Type(), ht.Hint().Type())
	t.NotEqual(orig.Hint().Version(), ht.Hint().Version())

	b, err := enc.Marshal(ht)
	t.NoError(err)

	hinter, err := enc.Decode(b)
	t.NoError(err)

	uht, ok := hinter.(hinterJSONUnpacker)
	t.True(ok)

	t.True(ht.Hint().Equal(uht.Hint()))

	t.Equal(ht.a, uht.a)
	t.Equal(ht.b, uht.b)
}

func (t *testJSONEncoder) TestDecodeSlice() {
	enc := NewEncoder()

	htj := hinterJSONMarshaller{
		h: hint.NewHint(hint.Type("text"), "v1.2.3"),
	}
	htu := newHinterJSONUnpacker(
		hint.NewHint(hint.Type("unpack"), "v1.2.3"),
		"",
		0,
	)
	t.NoError(enc.Add(htj))
	t.NoError(enc.Add(htu))

	ht0 := hinterJSONMarshaller{h: htj.Hint(), a: "A", b: 44}
	ht1 := newHinterJSONUnpacker(htu.Hint(), "a", 33)

	b, err := enc.Marshal([]interface{}{ht0, ht1})
	t.NoError(err)

	i, err := enc.DecodeSlice(b)
	t.NoError(err)

	t.Equal(2, len(i))

	uht0, ok := i[0].(hinterJSONMarshaller)
	t.True(ok)
	uht1, ok := i[1].(hinterJSONUnpacker)
	t.True(ok)

	t.Equal(ht0.a, uht0.a)
	t.Equal(ht0.b, uht0.b)

	t.Equal(ht1.a, uht1.a)
	t.Equal(ht1.b, uht1.b)
}

func (t *testJSONEncoder) TestDecodeMap() {
	enc := NewEncoder()

	htj := hinterJSONMarshaller{
		h: hint.NewHint(hint.Type("text"), "v1.2.3"),
	}
	htu := newHinterJSONUnpacker(
		hint.NewHint(hint.Type("unpack"), "v1.2.3"),
		"",
		0,
	)
	t.NoError(enc.Add(htj))
	t.NoError(enc.Add(htu))

	ht0 := hinterJSONMarshaller{h: htj.Hint(), a: "A", b: 44}
	ht1 := newHinterJSONUnpacker(htu.Hint(), "a", 33)

	b, err := enc.Marshal(map[string]interface{}{
		"ht0": ht0,
		"ht1": ht1,
	})
	t.NoError(err)

	i, err := enc.DecodeMap(b)
	t.NoError(err)

	t.Equal(2, len(i))

	uht0, ok := i["ht0"].(hinterJSONMarshaller)
	t.True(ok)
	uht1, ok := i["ht1"].(hinterJSONUnpacker)
	t.True(ok)

	t.Equal(ht0.a, uht0.a)
	t.Equal(ht0.b, uht0.b)

	t.Equal(ht1.a, uht1.a)
	t.Equal(ht1.b, uht1.b)
}

func (t *testJSONEncoder) TestDecodeKeepHint() {
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

func TestJSONEncoder(t *testing.T) {
	suite.Run(t, new(testJSONEncoder))
}
