package state

import (
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
)

type testFixedTreeNodeEncode struct {
	suite.Suite
	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testFixedTreeNodeEncode) SetupSuite() {
	t.encs = encoder.NewEncoders()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.TestAddHinter(FixedTreeNodeHinter)
}

func (t *testFixedTreeNodeEncode) TestMake() {
	no := NewFixedTreeNode(33, util.UUID().Bytes())
	no.BaseHinter = hint.NewBaseHinter(hint.NewHint(FixedTreeNodeType, "v0.0.9"))

	var raw []byte
	raw, err := t.enc.Marshal(no)
	t.NoError(err)

	hinter, err := t.enc.Decode(raw)
	t.NoError(err)

	uno, ok := hinter.(FixedTreeNode)
	t.True(ok)

	t.True(no.Hint().Equal(uno.Hint()))
	t.True(no.Equal(uno))
}

func TestFixedTreeNodeEncodeJSON(t *testing.T) {
	b := new(testFixedTreeNodeEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestFixedTreeNodeEncodeBSON(t *testing.T) {
	b := new(testFixedTreeNodeEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
