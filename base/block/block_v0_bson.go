package block

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (bm BlockV0) MarshalBSON() ([]byte, error) {
	m := bson.M{
		"manifest":  bm.ManifestV0,
		"consensus": bm.ci,
	}

	if bm.operationsTree.Len() > 0 {
		m["operations_tree"] = bm.operationsTree
	}

	if len(bm.operations) > 0 {
		m["operations"] = bm.operations
	}

	if bm.statesTree.Len() > 0 {
		m["states_tree"] = bm.statesTree
	}

	if len(bm.states) > 0 {
		m["states"] = bm.states
	}

	return bsonenc.Marshal(bsonenc.MergeBSONM(bsonenc.NewHintedDoc(bm.Hint()), m))
}

type BlockV0UnpackBSON struct {
	MF  bson.Raw `bson:"manifest"`
	CI  bson.Raw `bson:"consensus"`
	OPT bson.Raw `bson:"operations_tree,omitempty"`
	OP  bson.Raw `bson:"operations,omitempty"`
	STT bson.Raw `bson:"states_tree,omitempty"`
	ST  bson.Raw `bson:"states,omitempty"`
}

func (bm *BlockV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var um BlockV0UnpackBSON
	if err := enc.Unmarshal(b, &um); err != nil {
		return err
	}

	return bm.unpack(enc, um.MF, um.CI, um.OPT, um.OP, um.STT, um.ST)
}
