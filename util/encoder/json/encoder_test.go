package jsonenc

import (
	"fmt"
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type hinterDefault struct {
	h hint.Hint
	A string
	B int
}

func (ht hinterDefault) Hint() hint.Hint {
	return ht.h
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
	h hint.Hint
	a string
	b int
}

func (ht hinterJSONUnpacker) Hint() hint.Hint {
	return ht.h
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

type hinterTextMarshaller struct {
	h hint.Hint
	A int
	B int
}

func (ht hinterTextMarshaller) Hint() hint.Hint {
	return ht.h
}

func (ht hinterTextMarshaller) String() string {
	return hint.NewHintedString(ht.Hint(), fmt.Sprintf("%d-%d", ht.A, ht.B)).String()
}

func (ht hinterTextMarshaller) MarshalText() ([]byte, error) {
	return []byte(ht.String()), nil
}

func (ht *hinterTextMarshaller) UnmarshalText(b []byte) error {
	var ua, ub int
	n, err := fmt.Sscanf(string(b)+"\n", "%d-%d", &ua, &ub)

	if err != nil {
		return err
	} else if n != 2 {
		return xerrors.Errorf("something missed")
	}

	ht.A = ua
	ht.B = ub

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

	ht := hinterDefault{
		h: hint.NewHint(hint.Type("findme"), "v1.2.3"),
		A: "A", B: 33,
	}
	t.NoError(enc.Add(ht))

	// add again
	err := enc.Add(ht)
	t.Contains(err.Error(), "already added")
}

func (t *testJSONEncoder) TestDecodeUnknown() {
	enc := NewEncoder()

	ht := hinterDefault{
		h: hint.NewHint(hint.Type("findme"), "v1.2.3"),
		A: "A", B: 33,
	}

	b, err := enc.Marshal(ht)
	t.NoError(err)

	_, err = enc.Decode(b)
	t.True(xerrors.Is(err, util.NotFoundError))
}

func (t *testJSONEncoder) TestDecodeDefault() {
	enc := NewEncoder()

	ht := hinterDefault{
		h: hint.NewHint(hint.Type("findme"), "v1.2.3"),
	}
	t.NoError(enc.Add(ht))

	ht.A = "A"
	ht.B = 33

	b, err := enc.Marshal(ht)
	t.NoError(err)

	hinter, err := enc.Decode(b)
	t.NoError(err)

	uht, ok := hinter.(hinterDefault)
	t.True(ok)

	t.Equal(ht.A, uht.A)
	t.Equal(ht.B, uht.B)
}

func (t *testJSONEncoder) TestDecodeTextMarshaller() {
	enc := NewEncoder()

	ht := hinterTextMarshaller{
		h: hint.NewHint(hint.Type("findme"), "v1.2.3"),
	}
	t.NoError(enc.Add(ht))

	ht.A = 22
	ht.B = 33

	b, err := enc.Marshal(ht)
	t.NoError(err)

	hinter, err := enc.Decode(b)
	t.NoError(err)

	uht, ok := hinter.(hinterTextMarshaller)
	t.True(ok)

	t.Equal(ht.A, uht.A)
	t.Equal(ht.B, uht.B)
}

func (t *testJSONEncoder) TestDecodeJSONUnmarshaller() {
	enc := NewEncoder()

	ht := hinterJSONMarshaller{
		h: hint.NewHint(hint.Type("findme"), "v1.2.3"),
	}
	t.NoError(enc.Add(ht))

	ht.a = "fa"
	ht.b = 33

	b, err := enc.Marshal(ht)
	t.NoError(err)

	hinter, err := enc.Decode(b)
	t.NoError(err)

	uht, ok := hinter.(hinterJSONMarshaller)
	t.True(ok)

	t.Equal(ht.a, uht.a)
	t.Equal(ht.b, uht.b)
}

func (t *testJSONEncoder) TestDecodeJSONUnpacker() {
	enc := NewEncoder()

	ht := hinterJSONUnpacker{
		h: hint.NewHint(hint.Type("findme"), "v1.2.3"),
	}
	t.NoError(enc.Add(ht))

	ht.a = "fa"
	ht.b = 33

	b, err := enc.Marshal(ht)
	t.NoError(err)

	hinter, err := enc.Decode(b)
	t.NoError(err)

	uht, ok := hinter.(hinterJSONUnpacker)
	t.True(ok)

	t.Equal(ht.a, uht.a)
	t.Equal(ht.b, uht.b)
}

func (t *testJSONEncoder) TestDecodeWitHint() {
	enc := NewEncoder()

	htt := hinterTextMarshaller{
		h: hint.NewHint(hint.Type("text"), "v1.2.3"),
	}
	htj := hinterJSONUnpacker{
		h: hint.NewHint(hint.Type("unpack"), "v1.2.3"),
	}
	t.NoError(enc.Add(htt))
	t.NoError(enc.Add(htj))

	htj.a = "fa"
	htj.b = 33

	b, err := enc.Marshal(htj)
	t.NoError(err)

	_, err = enc.DecodeWithHint(b, htt.Hint())
	t.Contains(err.Error(), "failed to decode")
}

func (t *testJSONEncoder) TestDecodeSlice() {
	enc := NewEncoder()

	htj := hinterJSONMarshaller{
		h: hint.NewHint(hint.Type("text"), "v1.2.3"),
	}
	htu := hinterJSONUnpacker{
		h: hint.NewHint(hint.Type("unpack"), "v1.2.3"),
	}
	t.NoError(enc.Add(htj))
	t.NoError(enc.Add(htu))

	ht0 := hinterJSONMarshaller{h: htj.Hint(), a: "A", b: 44}
	ht1 := hinterJSONUnpacker{h: htu.Hint(), a: "a", b: 33}

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
	htu := hinterJSONUnpacker{
		h: hint.NewHint(hint.Type("unpack"), "v1.2.3"),
	}
	t.NoError(enc.Add(htj))
	t.NoError(enc.Add(htu))

	ht0 := hinterJSONMarshaller{h: htj.Hint(), a: "A", b: 44}
	ht1 := hinterJSONUnpacker{h: htu.Hint(), a: "a", b: 33}

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

func TestJSONEncoder(t *testing.T) {
	suite.Run(t, new(testJSONEncoder))
}
