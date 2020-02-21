package valuehash

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/hint"
)

type testDummy struct {
	suite.Suite
}

func (t *testDummy) SetupTest() {
	_ = hint.RegisterType(Dummy{}.Hint().Type(), "dummy")
}

func (t *testDummy) TestEmpty() {
	s := Dummy{}
	err := s.IsValid(nil)
	t.Contains(err.Error(), "empty")
}

func (t *testDummy) TestNil() {
	s := NewDummy(nil)
	err := s.IsValid(nil)
	t.Contains(err.Error(), "empty")
}

func (t *testDummy) TestNew() {
	dm := NewDummy(nil)
	t.Implements((*Hash)(nil), dm)
	t.Equal(0, dm.Size())

	initial := dm.Bytes()

	b := []byte("showme")
	dm = NewDummy(b)

	t.NotEqual(initial, dm.Bytes())

	newdm := NewDummy(b)

	t.Equal(dm.Bytes(), newdm.Bytes())
	t.Equal(len(b), newdm.Size())
	t.Equal(b, newdm.Bytes())
}

func (t *testDummy) TestJSONMarshal() {
	b := []byte("killme")
	dm := NewDummy(b)

	{
		b, err := MarshalJSON(dm)
		t.NoError(err)

		var jh JSONHash
		t.NoError(err, json.Unmarshal(b, &jh))

		t.Equal(dm.Hint(), jh.JSONPackHintedHead.H)
		t.Equal(dm.Bytes(), jh.Bytes())
	}

	{
		b, err := json.Marshal(dm)
		t.NoError(err)

		var jh JSONHash
		t.NoError(err, json.Unmarshal(b, &jh))

		t.Equal(dm.Hint(), jh.JSONPackHintedHead.H)
		t.Equal(dm.Bytes(), jh.Bytes())
	}
}

func TestDummy(t *testing.T) {
	suite.Run(t, new(testDummy))
}
