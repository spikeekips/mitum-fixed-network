package state

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/util"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var dummyHintedValueHint = hint.MustHintWithType(hint.Type{0xff, 0x60}, "0.0.1", "dummy-hinted-value")

type dummyNotHinted struct {
	v int
}

type dummyNotByter struct {
	dummyNotHinted
}

func (dv dummyNotByter) Hint() hint.Hint {
	return dummyHintedValueHint
}

type dummyNotHasher struct {
	dummyNotByter
}

func (dv dummyNotHasher) Bytes() []byte {
	return util.IntToBytes(dv.v)
}

type dummy struct {
	dummyNotHasher
}

func (dv dummy) Hash() valuehash.Hash {
	return valuehash.NewSHA256(dv.Bytes())
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
	t.Contains(err.Error(), "not hint.Hinter")
}

func (t *testStateHintedValue) TestNewNotByter() {
	v := dummyNotByter{}
	v.v = 1

	_, err := NewHintedValue(v)
	t.Contains(err.Error(), "not util.Byter")
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
	t.Equal(dv.Bytes(), v.Bytes())
}

func TestStateHintedValue(t *testing.T) {
	suite.Run(t, new(testStateHintedValue))
}
