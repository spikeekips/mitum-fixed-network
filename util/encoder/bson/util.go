package bsonenc

import (
	"github.com/spikeekips/mitum/util/hint"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

func Marshal(i interface{}) ([]byte, error) {
	return bson.Marshal(i)
}

func MarshalValue(i interface{}) (bsontype.Type, []byte, error) {
	return bson.MarshalValue(i)
}

func Unmarshal(b []byte, i interface{}) error {
	return bson.Unmarshal(b, i)
}

func MergeBSONM(a bson.M, b ...bson.M) bson.M {
	for _, c := range b {
		for k, v := range c {
			a[k] = v
		}
	}

	return a
}

type PackHintedHead struct {
	H hint.Hint `bson:"_hint"`
}

func NewHintedDoc(h hint.Hint) bson.M {
	return bson.M{"_hint": h}
}
