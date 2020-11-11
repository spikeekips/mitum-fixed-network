package policy

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

func (po PolicyV0) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(po.Hint()),
		bson.M{
			"number_of_acting_suffrage_nodes": po.numberOfActingSuffrageNodes,
			"max_operations_in_seal":          po.maxOperationsInSeal,
			"max_operations_in_proposal":      po.maxOperationsInProposal,
		},
	))
}

type PolicyV0UnpackBSON struct {
	NS uint `bson:"number_of_acting_suffrage_nodes"`
	MS uint `bson:"max_operations_in_seal"`
	MP uint `bson:"max_operations_in_proposal"`
}

func (po *PolicyV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var up PolicyV0UnpackBSON
	if err := enc.Unmarshal(b, &up); err != nil {
		return err
	}

	return po.unpack(up.NS, up.MS, up.MP)
}
