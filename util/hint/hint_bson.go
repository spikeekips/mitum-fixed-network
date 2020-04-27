package hint

import (
	"go.mongodb.org/mongo-driver/bson"
)

type hintBSON struct {
	T Type    `bson:"type"`
	V Version `bson:"version"`
}

func (ht Hint) MarshalBSON() ([]byte, error) {
	return bson.Marshal(hintBSON{
		T: ht.t,
		V: ht.version,
	})
}

func (ht *Hint) UnmarshalBSON(b []byte) error {
	var h hintBSON
	if err := bson.Unmarshal(b, &h); err != nil {
		return err
	}

	ht.t = h.T
	ht.version = h.V

	return nil
}
