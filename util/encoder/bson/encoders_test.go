package bsonenc

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
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

	t.NoError(encs.AddHinter(sh0{}))

	s0, err := encs.Hinter((sh0{}).Hint().Type(), (sh0{}).Hint().Version())
	t.NoError(err)
	t.NotNil(s0)
	t.NoError((sh0{}).Hint().IsCompatible(s0.Hint()))
	t.True((sh0{}).Hint().Equal(s0.Hint()))
}

func (t *testEncodersWithBSON) TestDecodeByHint() {
	encs := encoder.NewEncoders()
	be := NewEncoder()
	t.NoError(encs.AddEncoder(be))

	s := sh0{B: util.UUID().String()}

	b, err := be.Marshal(MergeBSONM(
		NewHintedDoc(s.Hint()),
		bson.M{
			"B": s.B,
		},
	))

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

func TestEncodersWithBSON(t *testing.T) {
	suite.Run(t, new(testEncodersWithBSON))
}
