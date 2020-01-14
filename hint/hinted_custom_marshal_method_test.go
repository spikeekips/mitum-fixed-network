package hint

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type customMarshalHinted struct {
	A int
	B string
}

func (mh customMarshalHinted) Hint() Hint {
	h, _ := NewHint(Type([2]byte{0xff, 0x12}), "0.0.3")
	return h
}

func (mh customMarshalHinted) marshalJSON() ([]byte, error) {
	return jsoni.Marshal(map[string]interface{}{
		"c": mh.A,
		"d": mh.B,
	})
}

func (mh customMarshalHinted) MarshalJSON() ([]byte, error) {
	b, err := mh.marshalJSON()
	if err != nil {
		return nil, err
	}
	hb, err := jsoni.Marshal(mh.Hint())
	if err != nil {
		return nil, err
	}

	var n []byte
	n = append(n, b[:1]...)
	n = append(n, []byte(`"_hint":`)...)
	n = append(n, hb...)
	n = append(n, ',')
	n = append(n, b[1:]...)

	return n, nil
}

func (mh *customMarshalHinted) unmarshalJSON(b []byte) error {
	var m map[string]interface{}

	if err := jsoni.Unmarshal(b, &m); err != nil {
		return err
	}

	mh.A = int(m["c"].(float64))
	mh.B = m["d"].(string)

	return nil
}

func (mh *customMarshalHinted) UnmarshalJSON(b []byte) error {
	return mh.unmarshalJSON(b)
}

type testCustomMarshalHinted struct {
	suite.Suite
}

func (t *testCustomMarshalHinted) TestNew() {
	mh := customMarshalHinted{A: 33, B: "findme"}

	t.Implements((*Hinter)(nil), mh)
}

func (t *testCustomMarshalHinted) TestHintFromJSONMarshaled() {
	mh := customMarshalHinted{A: 33, B: "findme"}

	_ = RegisterType(mh.Hint().Type(), "0xff12-v0.0.3")

	b, err := jsoni.Marshal(mh)
	t.NoError(err)
	t.NotNil(b)

	uh, err := HintFromJSONMarshaled(b)
	t.NoError(err)

	t.Equal(mh.Hint().Type(), uh.Type())
	t.Equal(mh.Hint().Version(), uh.Version())
}

func TestCustomMarshalHinted(t *testing.T) {
	suite.Run(t, new(testCustomMarshalHinted))
}
