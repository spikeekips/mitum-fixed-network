package node

import (
	"errors"
	"io"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
)

type testNodeEncode struct {
	suite.Suite
	Enc     encoder.Encoder
	Hinters []hint.Hinter
	Encode  func() (interface{}, []byte)
	Decode  func([]byte) (interface{}, error)
	Compare func(interface{}, interface{})
}

func newTestNodeEncode() *testNodeEncode {
	s := new(testNodeEncode)

	s.Hinters = []hint.Hinter{
		BaseV0Hinter,
		base.StringAddressHinter,
		key.BasePublickey{},
	}
	s.Decode = func(b []byte) (interface{}, error) {
		return s.Enc.Decode(b)
	}
	s.Compare = func(a, b interface{}) {
		an, ok := a.(base.Node)
		s.True(ok)
		bn, ok := b.(base.Node)
		s.True(ok)

		s.NoError(an.IsValid(nil))
		s.NoError(bn.IsValid(nil))

		s.True(an.Address().Equal(bn.Address()))
		s.True(an.Publickey().Equal(bn.Publickey()))
	}

	return s
}

func (t *testNodeEncode) SetupSuite() {
	for i := range t.Hinters {
		_ = t.Enc.Add(t.Hinters[i])
	}
}

func (t *testNodeEncode) TestDecode() {
	a, b := t.Encode()

	ua, err := t.Decode(b)
	if err != nil {
		return
	}

	t.Compare(a, ua)
}

func testNilNodeEncode() *testNodeEncode {
	s := newTestNodeEncode()

	s.Encode = func() (interface{}, []byte) {
		b, err := s.Enc.Marshal(nil)
		s.NoError(err)

		return nil, b
	}
	s.Decode = func(b []byte) (interface{}, error) {
		i, err := s.Enc.Decode(b)
		return i, err
	}
	s.Compare = func(a, b interface{}) {
		s.Nil(a)
		s.Nil(b)
	}

	return s
}

func testRemoteNodeEncode() *testNodeEncode {
	s := newTestNodeEncode()
	s.Encode = func() (interface{}, []byte) {
		a := NewRemote(base.RandomStringAddress(), key.NewBasePrivatekey().Publickey())

		b, err := s.Enc.Marshal(a)
		s.NoError(err)

		return a, b
	}

	return s
}

func testLocalNodeEncode() *testNodeEncode {
	s := newTestNodeEncode()
	s.Encode = func() (interface{}, []byte) {
		a := NewLocal(base.RandomStringAddress(), key.NewBasePrivatekey())

		b, err := s.Enc.Marshal(a)
		s.NoError(err)

		return a, b
	}

	return s
}

func TestLocalNodeEncodeJSON(t *testing.T) {
	s := testLocalNodeEncode()
	s.Enc = jsonenc.NewEncoder()

	suite.Run(t, s)
}

func TestRemoteNodeEncodeJSON(t *testing.T) {
	s := testRemoteNodeEncode()
	s.Enc = jsonenc.NewEncoder()

	suite.Run(t, s)
}

func TestNilNodeEncodeJSON(t *testing.T) {
	s := testNilNodeEncode()
	s.Enc = jsonenc.NewEncoder()

	suite.Run(t, s)
}

func TestRemoteNodeEncodeBSON(t *testing.T) {
	s := testRemoteNodeEncode()
	s.Enc = bsonenc.NewEncoder()

	suite.Run(t, s)
}

func TestLocalNodeEncodeBSON(t *testing.T) {
	s := testLocalNodeEncode()
	s.Enc = bsonenc.NewEncoder()

	suite.Run(t, s)
}

func TestNilNodeEncodeBSON(t *testing.T) {
	s := testNilNodeEncode()
	s.Enc = bsonenc.NewEncoder()
	s.Encode = func() (interface{}, []byte) {
		b, err := s.Enc.Marshal(struct {
			A base.Node
		}{})
		s.NoError(err)

		return nil, b
	}
	s.Decode = func(b []byte) (interface{}, error) {
		var u struct {
			A bson.Raw
		}

		s.NoError(s.Enc.Unmarshal(b, &u))

		var un BaseV0
		err := un.UnpackBSON(u.A, s.Enc.(*bsonenc.Encoder))
		s.True(errors.Is(err, io.EOF))

		return un, nil
	}
	s.Compare = func(a, _ interface{}) {
		s.Nil(a)
	}

	suite.Run(t, s)
}
