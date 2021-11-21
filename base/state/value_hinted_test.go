package state

import (
	"testing"

	"github.com/spikeekips/mitum/util"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
)

var dummyHintedValueHint = hint.NewHint(hint.Type("dummy-hinted-value"), "v0.0.1")

type dummyNotHinted struct {
	v int
}

type dummyNotHasher struct {
	dummyNotHinted
}

func (dv dummyNotHasher) Hint() hint.Hint {
	return dummyHintedValueHint
}

type dummy struct {
	dummyNotHasher
}

func (dv dummy) Hash() valuehash.Hash {
	return valuehash.NewSHA256(util.IntToBytes(dv.v))
}

func (dv dummy) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		V int
	}{
		HintedHead: jsonenc.NewHintedHead(dv.Hint()),
		V:          dv.v,
	})
}

func (dv *dummy) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var u struct{ V int }
	if err := enc.Unmarshal(b, &u); err != nil {
		return err
	}

	dv.v = u.V

	return nil
}

func (dv dummy) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bson.M{
		"_hint": dv.Hint(),
		"value": dv.v,
	})
}

func (dv *dummy) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var u struct {
		V int `bson:"value"`
	}
	if err := enc.Unmarshal(b, &u); err != nil {
		return err
	}

	dv.v = u.V

	return nil
}

type testStateHintedValue struct {
	suite.Suite
}

func (t *testStateHintedValue) TestNewNotHinted() {
	dv := HintedValue{}
	_, err := dv.Set(dummyNotHinted{v: 1})
	t.Contains(err.Error(), "not Hinter")
}

func (t *testStateHintedValue) TestNewNotHasher() {
	v := dummyNotHasher{}
	v.v = 1

	_, err := NewHintedValue(v)
	t.Contains(err.Error(), "not valuehash.Hasher")
}

func (t *testStateHintedValue) TestNew() {
	v := dummy{}
	v.v = 1

	dv, err := NewHintedValue(v)
	t.NoError(err)

	t.Equal(dv.v, v)
	t.True(dv.Hash().Equal(v.Hash()))
}

func TestStateHintedValue(t *testing.T) {
	suite.Run(t, new(testStateHintedValue))
}
