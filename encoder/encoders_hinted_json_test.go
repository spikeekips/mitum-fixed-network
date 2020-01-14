package encoder

import (
	"encoding/json"
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
)

func (h0 sH0) EncodeJSON(hs *HintEncoder) (interface{}, error) {
	return &struct {
		JSONHinterHead
		sH0
	}{
		sH0: h0,
	}, nil
}

func (h0 *sH0) DecodeJSON(hs *HintEncoder, b []byte) error {
	var s struct {
		sH0
	}

	if err := hs.Encoder().Unmarshal(b, &s); err != nil {
		return err
	}

	h0.A = s.A

	return nil
}

func (h0 sH1) EncodeJSON(hs *HintEncoder) (interface{}, error) {
	return &struct {
		JSONHinterHead
		sH1
	}{
		sH1: h0,
	}, nil
}

func (h0 sH2) EncodeJSON(hs *HintEncoder) (interface{}, error) {
	b, err := hs.Encoder().Marshal(struct {
		JSONHinterHead
		sH2
	}{
		JSONHinterHead: NewJSONHinterHead(h0.Hint()),
		sH2:            h0,
	})

	return json.RawMessage(b), err
}

func (h0 sH3) EncodeJSON(hs *HintEncoder) (interface{}, error) {
	b, err := hs.Encoder().Marshal(struct {
		JSONHinterHead
		sH3
	}{
		JSONHinterHead: NewJSONHinterHead(h0.Hint()),
		sH3:            h0,
	})

	return []byte(b), err
}

func (h0 sH4) EncodeJSON(hs *HintEncoder) (interface{}, error) {
	return map[string]interface{}{
		"_hint": h0.Hint(),
		"A":     h0.A,
		"B":     h0.B,
	}, nil
}

type testEncodersHintedJSON struct {
	suite.Suite
	encs *Encoders
}

func (t *testEncodersHintedJSON) SetupSuite() {
	je := NewHintEncoder(JSON{})
	_ = hint.RegisterType(je.Hint().Type(), "json-encoder")

	_ = hint.RegisterType(sH0{}.Hint().Type(), "hinted-0")
	_ = hint.RegisterType(sH1{}.Hint().Type(), "hinted-1")
}

// Test encoding and decoding of JSONEncoder.
// - sH0 is analyzed.
func (t *testEncodersHintedJSON) TestEncodeHinterAnalyzed() {
	je := NewHintEncoder(JSON{})
	_, err := je.Analyze(sH0{})
	t.NoError(err)

	s := sH0{A: uuid.Must(uuid.NewV4(), nil).String()}
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sH0
	t.NoError(je.Decode(b, &us))

	t.Equal(s.A, us.A)
	t.True(s.Hint().Equal(us.Hint()))
}

// Test encoding and decoding of JSONEncoder.
// - s0 is not Hinter and it does not have JSONEncoder and JSONDecoder.
func (t *testEncodersHintedJSON) TestEncodeNotHinter() {
	je := NewHintEncoder(JSON{})

	s := s0{A: uuid.Must(uuid.NewV4(), nil).String()}
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us s0
	t.NoError(je.Decode(b, &us))

	t.Equal(s.A, us.A)
}

func (t *testEncodersHintedJSON) TestAnalyze() {
	je := NewHintEncoder(JSON{})

	{
		dh, err := je.Analyze(s0{})
		t.NoError(err)
		t.False(dh.IsHinter)
		t.False(dh.HasEncoder)
		t.False(dh.HasDecoder)
		t.NotNil(dh.encode)
		t.NotNil(dh.decode)
	}

	{
		dh, err := je.Analyze(sH0{})
		t.NoError(err)
		t.True(dh.IsHinter)
		t.True(dh.HasEncoder)
		t.True(dh.HasDecoder)
		t.NotNil(dh.encode)
		t.NotNil(dh.decode)
	}

	{
		dh, err := je.Analyze(sH1{})
		t.NoError(err)
		t.True(dh.IsHinter)
		t.True(dh.HasEncoder)
		t.False(dh.HasDecoder)
		t.NotNil(dh.encode)
		t.NotNil(dh.decode)
	}
}

// Test encoding and decoding of JSONEncoder.
// - sH0 is not analyzed.
func (t *testEncodersHintedJSON) TestEncodeHinterNotAnalyzed() {
	je := NewHintEncoder(JSON{})

	s := sH0{A: uuid.Must(uuid.NewV4(), nil).String()}
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sH0
	t.NoError(je.Decode(b, &us))

	t.Equal(s.A, us.A)
	t.True(s.Hint().Equal(us.Hint()))
}

// Encode instance, which don't have DecodeJSON
func (t *testEncodersHintedJSON) TestEncodeHinterNotDecode() {
	je := NewHintEncoder(JSON{})

	s := sH1{A: uuid.Must(uuid.NewV4(), nil).String(), b: 33}
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sH1
	t.NoError(je.Decode(b, &us))

	t.Equal(s.A, us.A)
	t.NotEqual(s.b, us.b)
	t.Zero(us.b)
	t.True(s.Hint().Equal(us.Hint()))
}

func (t *testEncodersHintedJSON) TestDecodeByHint() {
	je := NewHintEncoder(JSON{})

	s0 := sH0{A: uuid.Must(uuid.NewV4(), nil).String()}
	b, err := je.Encode(s0)
	t.NoError(err)
	t.NotNil(b)

	{ // before Encoders.Add()
		_, err := je.DecodeByHint(b)
		t.Contains(err.Error(), "encs is nil")
	}

	encs := NewEncoders()
	t.NoError(encs.Add(je))

	{ // before HintEncoder.AddHinter()
		_, err := je.DecodeByHint(b)
		t.Contains(err.Error(), "Hint not found")
	}
	t.NoError(encs.AddHinter(sH0{}))

	decoded, err := je.DecodeByHint(b)
	t.NoError(err)
	t.IsType(sH0{}, decoded)
	t.Equal(s0, decoded.(sH0))
}

// sH0 has almost same shape with s0. TestDecodeNoneHinterWithHinted will decode
// sh0 with the encoded of sH0.
func (t *testEncodersHintedJSON) TestDecodeNoneHinterWithHinted() {
	je := NewHintEncoder(JSON{})
	encs := NewEncoders()
	t.NoError(encs.Add(je))
	t.NoError(encs.AddHinter(sH0{}))

	s := sH0{A: uuid.Must(uuid.NewV4(), nil).String()}
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us s0
	t.NoError(je.Decode(b, &us))
	t.Equal(s.A, us.A)
}

func (t *testEncodersHintedJSON) TestEncodeJSONReturnTypes() {
	je := NewHintEncoder(JSON{})
	encs := NewEncoders()
	t.NoError(encs.Add(je))
	t.NoError(encs.AddHinter(sH2{}))

	va := uuid.Must(uuid.NewV4(), nil).String()
	vb := 33

	check := func(s interface{}) {
		b, err := je.Encode(s)
		t.NoError(err)
		t.NotNil(b)

		var m map[string]interface{}
		t.NoError(je.Encoder().Unmarshal(b, &m))

		t.Equal(va, m["A"])
		t.Equal(vb, int(m["B"].(float64)))

		var h JSONHinterHead
		t.NoError(je.Encoder().Unmarshal(b, &h))
		t.Equal(s.(hint.Hinter).Hint(), h.H)
	}

	for _, s := range []interface{}{
		sH2{A: va, B: vb},
		sH3{A: va, B: vb},
		sH4{A: va, B: vb},
	} {
		check(s)
	}
}

func (t *testEncodersHintedJSON) TestLoadHint() {
	je := NewHintEncoder(JSON{})

	s := sH0{A: uuid.Must(uuid.NewV4(), nil).String()}
	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	// same Hint
	ht, err := je.loadHint(b, sH0{}.Hint())
	t.NoError(err)
	t.Equal(sH0{}.Hint(), ht)

	cases := []struct {
		name string
		t    [2]byte
		v    string
		err  error
	}{
		{
			name: "same type and version",
			t:    (sH0{}).Hint().Type(),
			v:    (sH0{}).Hint().Version(),
		},
		{
			name: "different type",
			t:    hint.Type([2]byte{0xff, 0x21}),
			v:    (sH0{}).Hint().Version(),
			err:  hint.TypeDoesNotMatchError,
		},
		{
			name: "same type and high patch version",
			t:    (sH0{}).Hint().Type(),
			v:    "1.2.4",
		},
		{
			name: "same type and lower patch version",
			t:    (sH0{}).Hint().Type(),
			v:    "1.2.2",
			err:  hint.VersionNotCompatibleError,
		},
		{
			name: "same type and high minor version",
			t:    (sH0{}).Hint().Type(),
			v:    "1.3.1",
		},
		{
			name: "same type and lower minor version",
			t:    (sH0{}).Hint().Type(),
			v:    "1.1.3",
			err:  hint.VersionNotCompatibleError,
		},
		{
			name: "same type and high major version",
			t:    (sH0{}).Hint().Type(),
			v:    "2.2.1",
			err:  hint.VersionNotCompatibleError,
		},
		{
			name: "same type and lower major version",
			t:    (sH0{}).Hint().Type(),
			v:    "0.2.3",
			err:  hint.VersionNotCompatibleError,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				target, err := hint.NewHint(hint.Type(c.t), c.v)
				t.NoError(err)

				h, err := je.loadHint(b, target)
				if c.err != nil {
					t.True(xerrors.Is(err, c.err), "%d: %v; %v != %v", i, c.name, c.err, err)
				} else if err != nil {
					t.NoError(err, "%d: %v; %v != %v", i, c.name, c.err, err)
				}

				if c.err == nil && err == nil {
					t.Equal(target, h)
				}
			},
		)
	}
}

func TestEncodersHintedJSON(t *testing.T) {
	suite.Run(t, new(testEncodersHintedJSON))
}
