package operation

import (
	"encoding/json"
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/tree"
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

	_ = t.encs.TestAddHinter(tree.BaseFixedTreeNode{})
	_ = t.encs.TestAddHinter(FixedTreeNode{})
	_ = t.encs.TestAddHinter(BaseReasonError{})
}

func (t *testFixedTreeNodeEncode) TestMake() {
	e := NewBaseReasonError("show me")
	e = e.SetData(map[string]interface{}{"a": 1})
	no := NewFixedTreeNodeWithHash(33, util.UUID().Bytes(), util.UUID().Bytes(), true, e)

	var raw []byte
	raw, err := t.enc.Marshal(no)
	t.NoError(err)

	hinter, err := t.enc.Decode(raw)
	t.NoError(err)

	uno, ok := hinter.(FixedTreeNode)
	t.True(ok)

	t.True(no.Equal(uno))

	ue := uno.Reason()

	ab, err := json.Marshal(e)
	t.NoError(err)
	bb, err := json.Marshal(ue)
	t.NoError(err)

	t.Equal(ab, bb)
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
