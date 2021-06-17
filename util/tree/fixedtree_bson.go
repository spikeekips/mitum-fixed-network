package tree

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

func (no BaseFixedTreeNode) M() map[string]interface{} {
	return map[string]interface{}{
		"index": no.index,
		"key":   no.key,
		"hash":  no.hash,
	}
}

func (no BaseFixedTreeNode) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(no.Hint()),
		no.M(),
	))
}

type BaseFixedTreeNodeBSONUnpacker struct {
	IN uint64 `bson:"index"`
	KY []byte `bson:"key"`
	HS []byte `bson:"hash"`
}

func (no *BaseFixedTreeNode) UnmarshalBSON(b []byte) error {
	var uno BaseFixedTreeNodeBSONUnpacker
	if err := bsonenc.Unmarshal(b, &uno); err != nil {
		return err
	}

	no.index = uno.IN
	no.key = uno.KY
	no.hash = uno.HS

	return nil
}

func (tr FixedTree) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(tr.Hint()),
		bson.M{
			"nodes": tr.nodes,
		},
	))
}

type FixedTreeBSONUnpacker struct {
	NS bson.Raw `bson:"nodes"`
}

func (tr *FixedTree) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var utr FixedTreeBSONUnpacker
	if err := enc.Unmarshal(b, &utr); err != nil {
		return err
	}

	return tr.unpack(enc, utr.NS)
}
