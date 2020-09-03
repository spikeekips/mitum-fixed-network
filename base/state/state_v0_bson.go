package state

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (st StateV0) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(st.Hint()),
		bson.M{
			"hash":           st.h,
			"key":            st.key,
			"value":          st.value,
			"previous_block": st.previousHeight,
			"height":         st.height,
			"operations":     st.operations,
		},
	))
}

type StateV0UnpackerBSON struct {
	H   valuehash.Bytes   `bson:"hash"`
	K   string            `bson:"key"`
	V   bson.Raw          `bson:"value"`
	PB  base.Height       `bson:"previous_block"`
	HT  base.Height       `bson:"height"`
	OPS []valuehash.Bytes `bson:"operations"`
}

func (st *StateV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ust StateV0UnpackerBSON
	if err := enc.Unmarshal(b, &ust); err != nil {
		return err
	}

	return st.unpack(enc, ust.H, ust.K, ust.V, ust.PB, ust.HT, ust.OPS)
}
