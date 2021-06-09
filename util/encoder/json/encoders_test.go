package jsonenc

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

type testEncodersWithJSON struct {
	suite.Suite
}

func (t *testEncodersWithJSON) TestAddEncoder() {
	encs := encoder.NewEncoders()

	je := NewEncoder()
	t.NoError(encs.AddEncoder(je))

	nje, err := encs.Encoder(je.Hint().Type(), je.Hint().Version())
	t.NoError(err)
	t.NoError(je.Hint().IsCompatible(nje.Hint()))
	t.True(je.Hint().Equal(nje.Hint()))
}

func (t *testEncodersWithJSON) TestAddHinter() {
	encs := encoder.NewEncoders()
	je := NewEncoder()
	t.NoError(encs.AddEncoder(je))

	t.NoError(encs.TestAddHinter(sh0{}))

	s0, err := encs.Compatible((sh0{}).Hint().Type(), (sh0{}).Hint().Version())
	t.NoError(err)
	t.NotNil(s0)
	t.NoError((sh0{}).Hint().IsCompatible(s0.Hint()))
	t.True((sh0{}).Hint().Equal(s0.Hint()))
}

func (t *testEncodersWithJSON) TestDecodeByHint() {
	encs := encoder.NewEncoders()
	je := NewEncoder()
	t.NoError(encs.AddEncoder(je))

	s := sh0{B: util.UUID().String()}

	b, err := Marshal(s)
	t.NoError(err)
	t.NotNil(b)

	{ // without AddHinter
		a, err := je.DecodeByHint(b)
		t.Empty(a)
		t.True(xerrors.Is(err, util.NotFoundError))
	}

	encs = encoder.NewEncoders()
	je = NewEncoder()
	t.NoError(encs.AddEncoder(je))

	t.NoError(encs.TestAddHinter(sh0{}))
	us, err := je.DecodeByHint(b)
	t.NoError(err)
	t.IsType(sh0{}, us)
	t.Equal(s.B, us.(sh0).B)
}

func TestEncodersWithJSON(t *testing.T) {
	suite.Run(t, new(testEncodersWithJSON))
}
