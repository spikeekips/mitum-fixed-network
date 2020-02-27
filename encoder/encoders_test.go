package encoder

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/util"
)

type testEncoders struct {
	suite.Suite
}

func (t *testEncoders) SetupSuite() {
	_ = hint.RegisterType(NewJSONEncoder().Hint().Type(), "1st-json-encoder")
	_ = hint.RegisterType(NewBSONEncoder().Hint().Type(), "1st-bson-encoder")
	_ = hint.RegisterType(NewRLPEncoder().Hint().Type(), "1st-rlp-encoder")
	_ = hint.RegisterType(sh0{}.Hint().Type(), "sh0")
}

func (t *testEncoders) TestAddEncoder() {
	encs := NewEncoders()

	je := NewJSONEncoder()
	t.NoError(encs.AddEncoder(je))

	nje, err := encs.Encoder(je.Hint().Type(), je.Hint().Version())
	t.NoError(err)
	t.NoError(je.Hint().IsCompatible(nje.Hint()))
	t.True(je.Hint().Equal(nje.Hint()))
}

func (t *testEncoders) TestAddHinter() {
	encs := NewEncoders()
	je := NewJSONEncoder()
	t.NoError(encs.AddEncoder(je))

	t.NoError(encs.AddHinter(sh0{}))

	s0, err := encs.Hinter((sh0{}).Hint().Type(), (sh0{}).Hint().Version())
	t.NoError(err)
	t.NotNil(s0)
	t.NoError((sh0{}).Hint().IsCompatible(s0.Hint()))
	t.True((sh0{}).Hint().Equal(s0.Hint()))
}

func (t *testEncoders) TestJSONDecodeByHint() {
	encs := NewEncoders()
	je := NewJSONEncoder()
	t.NoError(encs.AddEncoder(je))

	s := sh0{B: util.UUID().String()}

	b, err := je.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	{ // without AddHinter
		a, err := je.DecodeByHint(b)
		t.Empty(a)
		t.True(xerrors.Is(err, hint.HintNotFoundError))
	}

	t.NoError(encs.AddHinter(sh0{}))
	us, err := je.DecodeByHint(b)
	t.NoError(err)
	t.IsType(sh0{}, us)
	t.Equal(s.B, us.(sh0).B)
}

func (t *testEncoders) TestBSONDecodeByHint() {
	encs := NewEncoders()
	be := NewBSONEncoder()
	t.NoError(encs.AddEncoder(be))

	s := sh0{B: util.UUID().String()}

	b, err := be.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	{ // without AddHinter
		a, err := be.DecodeByHint(b)
		t.Empty(a)
		t.True(xerrors.Is(err, hint.HintNotFoundError))
	}

	t.NoError(encs.AddHinter(sh0{}))
	us, err := be.DecodeByHint(b)
	t.NoError(err)
	t.IsType(sh0{}, us)
	t.Equal(s.B, us.(sh0).B)
}

func (t *testEncoders) TestRLPDecodeByHint() {
	encs := NewEncoders()
	re := NewRLPEncoder()
	t.NoError(encs.AddEncoder(re))

	s := sh0{B: util.UUID().String()}

	b, err := re.Encode(s)
	t.NoError(err)
	t.NotNil(b)

	{ // without AddHinter
		a, err := re.DecodeByHint(b)
		t.Empty(a)
		t.True(xerrors.Is(err, hint.HintNotFoundError))
	}

	t.NoError(encs.AddHinter(sh0{}))
	us, err := re.DecodeByHint(b)
	t.NoError(err)
	t.IsType(sh0{}, us)
	t.Equal(s.B, us.(sh0).B)
}

func TestEncoders(t *testing.T) {
	suite.Run(t, new(testEncoders))
}
