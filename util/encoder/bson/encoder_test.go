package bsonenc

import (
	"fmt"
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
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

func (ht hinterDefault) MarshalBSON() ([]byte, error) {
	return bson.Marshal(MergeBSONM(NewHintedDoc(ht.h), bson.M{
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
	h hint.Hint
	a string
	b int
}

func (ht hinterBSONUnpacker) Hint() hint.Hint {
	return ht.h
}

func (ht hinterBSONUnpacker) MarshalBSON() ([]byte, error) {
	return bson.Marshal(MergeBSONM(NewHintedDoc(ht.h), bson.M{
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

func (ht hinterTextMarshaller) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, ht.String()), nil
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

type hinterEmbedText struct {
	H  hint.Hint
	HT hinterTextMarshaller
}

func (ht hinterEmbedText) Hint() hint.Hint {
	return ht.H
}

func (ht hinterEmbedText) MarshalBSON() ([]byte, error) {
	return bson.Marshal(MergeBSONM(NewHintedDoc(ht.H), bson.M{
		"HT": ht.HT,
	}))
}

func (ht *hinterEmbedText) UnpackBSON(b []byte, enc *Encoder) error {
	var u struct {
		H  bson.Raw
		HT bson.RawValue
	}

	if err := bson.Unmarshal(b, &u); err != nil {
		return err
	}

	i, err := enc.Decode(u.HT.Value)
	if err != nil {
		return err
	}

	j, ok := i.(hinterTextMarshaller)
	if !ok {
		return util.WrongTypeError.Errorf("expected hinterTextMarshaller, not %T", i)
	}

	ht.HT = j

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

	ht := hinterDefault{
		h: hint.NewHint(hint.Type("findme"), "v1.2.3"),
		A: "A", B: 33,
	}
	t.NoError(enc.Add(ht))

	// add again
	err := enc.Add(ht)
	t.Contains(err.Error(), "already added")
}

func (t *testBSONEncoder) TestDecodeUnknown() {
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

func (t *testBSONEncoder) TestDecodeDefault() {
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

func (t *testBSONEncoder) TestDecodeTextMarshaller() {
	enc := NewEncoder()

	ht := hinterTextMarshaller{
		h: hint.NewHint(hint.Type("findme"), "v1.2.3"),
	}
	t.NoError(enc.Add(ht))

	het := hinterEmbedText{
		H: hint.NewHint(hint.Type("showme"), "v1.2.3"),
	}
	t.NoError(enc.Add(het))

	ht.A = 22
	ht.B = 33
	het.HT = ht

	b, err := enc.Marshal(het)
	t.NoError(err)

	hinter, err := enc.Decode(b)
	t.NoError(err)

	uht, ok := hinter.(hinterEmbedText)
	t.True(ok)

	t.Equal(ht.A, uht.HT.A)
	t.Equal(ht.B, uht.HT.B)
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

	t.Equal(ht.a, uht.a)
	t.Equal(ht.b, uht.b)
}

func (t *testBSONEncoder) TestDecodeBSONUnpacker() {
	enc := NewEncoder()

	ht := hinterBSONUnpacker{
		h: hint.NewHint(hint.Type("findme"), "v1.2.3"),
	}
	t.NoError(enc.Add(ht))

	ht.a = "fa"
	ht.b = 33

	b, err := enc.Marshal(ht)
	t.NoError(err)

	hinter, err := enc.Decode(b)
	t.NoError(err)

	uht, ok := hinter.(hinterBSONUnpacker)
	t.True(ok)

	t.Equal(ht.a, uht.a)
	t.Equal(ht.b, uht.b)
}

func (t *testBSONEncoder) TestDecodeWitHint() {
	enc := NewEncoder()

	htt := hinterTextMarshaller{
		h: hint.NewHint(hint.Type("text"), "v1.2.3"),
	}
	htj := hinterBSONUnpacker{
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

func (t *testBSONEncoder) TestDecodeSlice() {
	enc := NewEncoder()

	htj := hinterBSONMarshaller{
		h: hint.NewHint(hint.Type("text"), "v1.2.3"),
	}
	htu := hinterBSONUnpacker{
		h: hint.NewHint(hint.Type("unpack"), "v1.2.3"),
	}

	hs := hinterSlice{
		H: hint.NewHint(hint.Type("slice"), "v1.2.3"),
	}
	t.NoError(enc.Add(htj))
	t.NoError(enc.Add(htu))
	t.NoError(enc.Add(hs))

	ht0 := hinterBSONMarshaller{h: htj.Hint(), a: "A", b: 44}
	ht1 := hinterBSONUnpacker{h: htu.Hint(), a: "a", b: 33}

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
	htu := hinterBSONUnpacker{
		h: hint.NewHint(hint.Type("unpack"), "v1.2.3"),
	}
	t.NoError(enc.Add(htj))
	t.NoError(enc.Add(htu))

	ht0 := hinterBSONMarshaller{h: htj.Hint(), a: "A", b: 44}
	ht1 := hinterBSONUnpacker{h: htu.Hint(), a: "a", b: 33}

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

func TestBSONEncoder(t *testing.T) {
	suite.Run(t, new(testBSONEncoder))
}
