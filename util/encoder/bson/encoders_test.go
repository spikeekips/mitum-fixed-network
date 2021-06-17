package bsonenc

import (
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testEncodersWithBSON struct {
	suite.Suite
}

func (t *testEncodersWithBSON) TestAddEncoder() {
	encs := encoder.NewEncoders()

	je := NewEncoder()
	t.NoError(encs.AddEncoder(je))

	nje, err := encs.Encoder(je.Hint().Type(), je.Hint().Version())
	t.NoError(err)
	t.NoError(je.Hint().IsCompatible(nje.Hint()))
	t.True(je.Hint().Equal(nje.Hint()))
}

func (t *testEncodersWithBSON) TestAddHinter() {
	encs := encoder.NewEncoders()
	je := NewEncoder()
	t.NoError(encs.AddEncoder(je))

	ht := hinterDefault{
		h: hint.NewHint(hint.Type("findme"), "v1.2.3"),
		A: "A", B: 33,
	}

	t.NoError(encs.TestAddHinter(ht))

	s0, err := encs.Compatible((ht).Hint())
	t.NoError(err)
	t.NotNil(s0)
	t.NoError((ht).Hint().IsCompatible(s0.Hint()))
	t.True((ht).Hint().Equal(s0.Hint()))
}

func (t *testEncodersWithBSON) TestDecode() {
	encs := encoder.NewEncoders()
	be := NewEncoder()
	t.NoError(encs.AddEncoder(be))

	ht := hinterDefault{
		h: hint.NewHint(hint.Type("findme"), "v1.2.3"),
		A: "A", B: 33,
	}

	b, err := be.Marshal(ht)

	t.NoError(err)
	t.NotNil(b)

	{ // without AddHinter
		a, err := be.Decode(b)
		t.Empty(a)
		t.True(xerrors.Is(err, util.NotFoundError))
	}

	encs = encoder.NewEncoders()
	be = NewEncoder()
	t.NoError(encs.AddEncoder(be))

	t.NoError(encs.TestAddHinter(ht))
	hinter, err := be.Decode(b)
	t.NoError(err)

	uht, ok := hinter.(hinterDefault)
	t.True(ok)

	t.Equal(ht.A, uht.A)
	t.Equal(ht.B, uht.B)
}

func TestEncodersWithBSON(t *testing.T) {
	suite.Run(t, new(testEncodersWithBSON))
}
