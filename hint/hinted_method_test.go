package hint

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type methodHinted struct {
	A int
	B string
}

func (mh methodHinted) Hint() Hint {
	h, _ := NewHint(Type([2]byte{0xff, 0x11}), "0.0.2")
	return h
}

func (mh methodHinted) MarshalJSON() ([]byte, error) {
	return jsoni.Marshal(struct {
		H Hint `json:"_hint"`
		A int
		B string
	}{
		H: mh.Hint(),
		A: mh.A,
		B: mh.B,
	})
}

type testMethodHinted struct {
	suite.Suite
}

func (t *testMethodHinted) TestNew() {
	mh := methodHinted{A: 33, B: "findme"}

	t.Implements((*Hinter)(nil), mh)
}

func (t *testMethodHinted) TestHintFromJSONMarshaled() {
	mh := methodHinted{A: 33, B: "findme"}

	_ = RegisterType(mh.Hint().Type(), "0xff11-v0.0.2")

	b, err := jsoni.Marshal(mh)
	t.NoError(err)
	t.NotNil(b)

	uh, err := HintFromJSONMarshaled(b)
	t.NoError(err)

	t.Equal(mh.Hint().Type(), uh.Type())
	t.Equal(mh.Hint().Version(), uh.Version())
}

func TestMethodHinted(t *testing.T) {
	suite.Run(t, new(testMethodHinted))
}
