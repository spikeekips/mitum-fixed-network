package state

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
)

func (stav StateV0AVLNode) MarshalBSON() ([]byte, error) {
	return bsonencoder.Marshal(bsonencoder.MergeBSONM(
		bsonencoder.NewHintedDoc(stav.Hint()),
		bson.M{
			"hash":       stav.h,
			"key":        stav.Key(),
			"height":     stav.height,
			"left_key":   stav.left,
			"left_hash":  stav.leftHash,
			"right_key":  stav.right,
			"right_hash": stav.rightHash,
			"state":      stav.state,
		},
	))
}

type StateV0AVLNodeUnpackerBSON struct {
	H   []byte   `bson:"hash"`
	HT  int16    `bson:"height"`
	LF  []byte   `bson:"left_key"`
	LFH []byte   `bson:"left_hash"`
	RG  []byte   `bson:"right_key"`
	RGH []byte   `bson:"right_hash"`
	ST  bson.Raw `bson:"state"`
}

func (stav *StateV0AVLNode) UnpackBSON(b []byte, enc *bsonencoder.Encoder) error {
	var us StateV0AVLNodeUnpackerBSON
	if err := enc.Unmarshal(b, &us); err != nil {
		return err
	}

	return stav.unpack(enc, us.H, us.HT, us.LF, us.LFH, us.RG, us.RGH, us.ST)
}
