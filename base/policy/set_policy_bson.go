package policy

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (spo SetPolicyV0) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(spo.Hint()),
		bson.M{
			"hash":       spo.h,
			"fact_signs": spo.fs,
			"token":      spo.token,
			"policy":     spo.SetPolicyFactV0.PolicyV0,
		},
	))
}

type SetPolicyV0UnpackBSON struct {
	H  valuehash.Bytes `bson:"hash"`
	FS []bson.Raw      `bson:"fact_signs"`
	TK []byte          `bson:"token"`
	PO bson.Raw        `bson:"policy"`
}

func (spo *SetPolicyV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uspo SetPolicyV0UnpackBSON
	if err := enc.Unmarshal(b, &uspo); err != nil {
		return err
	}

	fs := make([][]byte, len(uspo.FS))
	for i := range uspo.FS {
		fs[i] = uspo.FS[i]
	}

	return spo.unpack(enc, uspo.H, fs, uspo.TK, uspo.PO)
}
