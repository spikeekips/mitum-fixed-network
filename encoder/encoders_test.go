package encoder

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/hint"
)

type testEncoders struct {
	suite.Suite
}

func (t *testEncoders) SetupTest() {
	je := JSON{}
	_ = hint.RegisterType(je.Hint().Type(), "json-encoder-v0.1")
}

func (t *testEncoders) TestNew() {
	je := NewHintEncoder(JSON{})

	encoders := NewEncoders()
	t.NoError(encoders.Add(je))

	ec, err := encoders.HintEncoder(je.Hint().Type(), je.Hint().Version())
	t.NoError(err)
	t.NotNil(ec)
}

func (t *testEncoders) TestJSONDecodeHint() {
	encoders := NewEncoders()
	_ = encoders.Add(NewHintEncoder(JSON{}))

	ec, _ := encoders.HintEncoder(JSON{}.Hint().Type(), JSON{}.Hint().Version())

	ht, _ := hint.NewHint(hint.Type([2]byte{0xff, 0xf0}), "0.1")
	_ = hint.RegisterType(ht.Type(), "find me")

	data := struct {
		JSONHinterHead
		A string
		B int
	}{
		JSONHinterHead: NewJSONHinterHead(ht),
		A:              "findme",
		B:              33,
	}
	b, err := ec.Encoder().Marshal(data)
	t.NoError(err)

	eht, err := ec.Encoder().DecodeHint(b)
	t.NoError(err)

	t.Equal(ht.Type(), eht.Type())
	t.Equal(ht.Version(), eht.Version())
}

func (t *testEncoders) TestJSONMarshal() {
	encoders := NewEncoders()
	_ = encoders.Add(NewHintEncoder(JSON{}))

	ec, _ := encoders.HintEncoder(JSON{}.Hint().Type(), JSON{}.Hint().Version())

	data := []string{"1", "2"}
	b, err := ec.Encoder().Marshal(data)
	t.NoError(err)

	var sl []string
	t.NoError(ec.Encoder().Unmarshal(b, &sl))

	t.Equal(data, sl)
}

func TestEncoders(t *testing.T) {
	suite.Run(t, new(testEncoders))
}
