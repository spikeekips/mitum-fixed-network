package block

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
)

func (bm ManifestV0) MarshalBSON() ([]byte, error) {
	m := bson.M{
		"hash":           bm.h,
		"height":         bm.height,
		"round":          bm.round,
		"proposal":       bm.proposal,
		"previous_block": bm.previousBlock,
		"created_at":     bm.createdAt,
	}
	if bm.operationsHash != nil {
		m["block_operations"] = bm.operationsHash
	}

	if bm.statesHash != nil {
		m["block_states"] = bm.statesHash
	}

	return bsonencoder.Marshal(bsonencoder.MergeBSONM(bsonencoder.NewHintedDoc(bm.Hint()), m))
}

type ManifestV0UnpackBSON struct {
	H  bson.Raw    `bson:"hash"`
	HT base.Height `bson:"height"`
	RD base.Round  `bson:"round"`
	PR bson.Raw    `bson:"proposal"`
	PB bson.Raw    `bson:"previous_block"`
	BO bson.Raw    `bson:"block_operations,omitempty"`
	BS bson.Raw    `bson:"block_states,omitempty"`
	CA time.Time   `bson:"created_at"`
}

func (bm *ManifestV0) UnpackBSON(b []byte, enc *bsonencoder.Encoder) error {
	var nbm ManifestV0UnpackBSON
	if err := enc.Unmarshal(b, &nbm); err != nil {
		return err
	}

	return bm.unpack(enc, nbm.H, nbm.HT, nbm.RD, nbm.PR, nbm.PB, nbm.BO, nbm.BS, nbm.CA)
}
