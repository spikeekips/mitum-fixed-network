package hint

import "go.mongodb.org/mongo-driver/bson"

type hintBSON struct {
	Type    Type   `bson:"t"`
	Version string `bson:"v"`
}

func (ht Hint) MarshalBSON() ([]byte, error) {
	return bson.Marshal(hintBSON{
		Type:    ht.t,
		Version: ht.version,
	})
}

func (ht *Hint) UnmarshalBSON(b []byte) error {
	var h hintBSON
	if err := bson.Unmarshal(b, &h); err != nil {
		return err
	}

	ht.t = h.Type
	ht.version = h.Version

	return nil
}
