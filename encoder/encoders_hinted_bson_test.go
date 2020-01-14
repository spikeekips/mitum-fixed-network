package encoder

import (
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
)

func (h0 sH0) EncodeBSON(_ *HintEncoder) (interface{}, error) {
	return h0, nil
}

func (h0 *sH0) DecodeBSON(hs *HintEncoder, b []byte) error {
	var s sH0
	if err := hs.Encoder().Unmarshal(b, &s); err != nil {
		return err
	}

	h0.A = s.A

	return nil
}

func (h0 sH2) EncodeBSON(hs *HintEncoder) (interface{}, error) {
	b, err := hs.Encoder().Marshal(struct {
		BSONHinterHead
		sH2
	}{
		BSONHinterHead: NewBSONHinterHead(h0.Hint()),
		sH2:            h0,
	})

	return bson.Raw(b), err
}

func (h0 sH3) EncodeBSON(hs *HintEncoder) (interface{}, error) {
	b, err := hs.Encoder().Marshal(struct {
		BSONHinterHead
		sH3
	}{
		BSONHinterHead: NewBSONHinterHead(h0.Hint()),
		sH3:            h0,
	})

	return []byte(b), err
}

func (h0 sH4) EncodeBSON(_ *HintEncoder) (interface{}, error) {
	return map[string]interface{}{
		"_hint": h0.Hint(),
		"A":     h0.A,
		"B":     h0.B,
	}, nil
}

type testEncodersHintedBSON struct {
	suite.Suite
	encs *Encoders
}

func (t *testEncodersHintedBSON) SetupSuite() {
	be := NewHintEncoder(BSON{})
	_ = hint.RegisterType(be.Hint().Type(), "bson-encoder")

	_ = hint.RegisterType(sH0{}.Hint().Type(), "hinted-0")
	_ = hint.RegisterType(sH1{}.Hint().Type(), "hinted-1")
}

// Test encoding and decoding of BSONEncoder.
// - sH0 is analyzed.
func (t *testEncodersHintedBSON) TestEncodeHinterAnalyzed() {
	be := NewHintEncoder(BSON{})
	_, err := be.Analyze(sH0{})
	t.NoError(err)

	s := sH0{A: uuid.Must(uuid.NewV4(), nil).String()}
	b, err := be.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sH0
	t.NoError(be.Decode(b, &us))

	t.Equal(s.A, us.A)
	t.True(s.Hint().Equal(us.Hint()))
}

// Test encoding and decoding of BSONEncoder.
// - s0 is not Hinter and it does not have BSONEncoder and BSONDecoder.
func (t *testEncodersHintedBSON) TestEncodeNotHinter() {
	be := NewHintEncoder(BSON{})

	s := s0{A: uuid.Must(uuid.NewV4(), nil).String()}
	b, err := be.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us s0
	t.NoError(be.Decode(b, &us))

	t.Equal(s.A, us.A)
}

func (t *testEncodersHintedBSON) TestAnalyze() {
	be := NewHintEncoder(BSON{})

	{
		dh, err := be.Analyze(s0{})
		t.NoError(err)
		t.False(dh.IsHinter)
		t.False(dh.HasEncoder)
		t.True(dh.HasDecoder)
		t.NotNil(dh.encode)
		t.NotNil(dh.decode)
	}

	{
		dh, err := be.Analyze(sH0{})
		t.NoError(err)
		t.True(dh.IsHinter)
		t.True(dh.HasEncoder)
		t.True(dh.HasDecoder)
		t.NotNil(dh.encode)
		t.NotNil(dh.decode)
	}

	{
		dh, err := be.Analyze(sH1{})
		t.NoError(err)
		t.True(dh.IsHinter)
		t.False(dh.HasEncoder)
		t.True(dh.HasDecoder)
		t.NotNil(dh.encode)
		t.NotNil(dh.decode)
	}
}

// Test encoding and decoding of BSONEncoder.
// - sH0 is not analyzed.
func (t *testEncodersHintedBSON) TestEncodeHinterNotAnalyzed() {
	be := NewHintEncoder(BSON{})

	s := sH0{A: uuid.Must(uuid.NewV4(), nil).String()}
	b, err := be.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sH0
	t.NoError(be.Decode(b, &us))

	t.Equal(s.A, us.A)
	t.True(s.Hint().Equal(us.Hint()))
}

// Encode instance, which don't have DecodeBSON
func (t *testEncodersHintedBSON) TestEncodeHinterNotDecode() {
	be := NewHintEncoder(BSON{})

	s := sH1{A: uuid.Must(uuid.NewV4(), nil).String(), b: 33}
	b, err := be.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us sH1
	t.NoError(be.Decode(b, &us))

	t.Equal(s.A, us.A)
	t.NotEqual(s.b, us.b)
	t.Zero(us.b)
	t.True(s.Hint().Equal(us.Hint()))
}

func (t *testEncodersHintedBSON) TestDecodeByHint() {
	be := NewHintEncoder(BSON{})

	s0 := sH0{A: uuid.Must(uuid.NewV4(), nil).String()}
	b, err := be.Encode(s0)
	t.NoError(err)
	t.NotNil(b)

	{ // before Encoders.Add()
		_, err := be.DecodeByHint(b)
		t.Contains(err.Error(), "encs is nil")
	}

	encs := NewEncoders()
	t.NoError(encs.Add(be))

	{ // before HintEncoder.AddHinter()
		_, err := be.DecodeByHint(b)
		t.Contains(err.Error(), "Hint not found")
	}
	t.NoError(encs.AddHinter(sH0{}))

	decoded, err := be.DecodeByHint(b)
	t.NoError(err)
	t.IsType(sH0{}, decoded)
	t.Equal(s0, decoded.(sH0))
}

// sH0 has almost same shape with s0. TestDecodeNoneHinterWithHinted fails to
// decode.
func (t *testEncodersHintedBSON) TestDecodeNoneHinterWithHinted() {
	be := NewHintEncoder(BSON{})
	encs := NewEncoders()
	t.NoError(encs.Add(be))
	t.NoError(encs.AddHinter(sH0{}))

	s := sH0{A: uuid.Must(uuid.NewV4(), nil).String()}
	b, err := be.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	var us s0
	t.NoError(be.Decode(b, &us))
	t.NotEqual(s.A, us.A)
}

func (t *testEncodersHintedBSON) TestLoadHint() {
	be := NewHintEncoder(BSON{})

	s := sH0{A: uuid.Must(uuid.NewV4(), nil).String()}
	b, err := be.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	// same Hint
	ht, err := be.loadHint(b, sH0{}.Hint())
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

				h, err := be.loadHint(b, target)
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

func TestEncodersHintedBSON(t *testing.T) {
	suite.Run(t, new(testEncodersHintedBSON))
}
