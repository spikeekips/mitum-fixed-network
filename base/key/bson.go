package key

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
)

func MarshalBSONKey(k hint.Hinter) ([]byte, error) {
	return bsonencoder.Marshal(bsonencoder.MergeBSONM(
		bsonencoder.NewHintedDoc(k.Hint()),
		bson.M{
			"key": k.(fmt.Stringer).String(),
		},
	))
}

type keyUnpackerBSON struct {
	HI hint.Hint `bson:"_hint"`
	K  string    `bson:"key"`
}

func UnmarshalBSONKey(b []byte) (hint.Hint, string, error) {
	var k keyUnpackerBSON
	if err := bsonencoder.Unmarshal(b, &k); err != nil {
		return hint.Hint{}, "", err
	}

	return k.HI, k.K, nil
}
