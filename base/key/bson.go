package key

import (
	"github.com/spikeekips/mitum/util/hint"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func marshalBSONStringKey(k Key) (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, hint.HintedString(k.Hint(), k.String())), nil
}
