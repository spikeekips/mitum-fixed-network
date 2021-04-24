package operation

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/tree"
	"go.mongodb.org/mongo-driver/bson"
)

func (no FixedTreeNode) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(no.Hint()),
		no.M(),
		bson.M{
			"in_state": no.inState,
			"reason":   no.reason,
		},
	))
}

type FixedTreeNodeBSONUnpacker struct {
	IS bool     `bson:"in_state"`
	RS bson.Raw `bson:"reason"`
}

func (no *FixedTreeNode) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubno tree.BaseFixedTreeNode
	if err := bsonenc.Unmarshal(b, &ubno); err != nil {
		return err
	}

	var uno FixedTreeNodeBSONUnpacker
	if err := bsonenc.Unmarshal(b, &uno); err != nil {
		return err
	}

	return no.unpack(enc, ubno, uno.IS, uno.RS)
}

func (e BaseReasonError) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(e.Hint()),
		bson.M{
			"msg":  e.Msg(),
			"data": e.data,
		},
	))
}

type BaseReasonErrorBSONUnpacker struct {
	MS string                 `bson:"msg"`
	DT map[string]interface{} `bson:"data"`
}

func (e *BaseReasonError) UnmarshalBSON(b []byte) error {
	var ue BaseReasonErrorBSONUnpacker
	if err := bsonenc.Unmarshal(b, &ue); err != nil {
		return err
	}

	e.NError = errors.NewError(ue.MS)
	e.msg = ue.MS
	e.data = ue.DT

	return nil
}
