package hint

import (
	"encoding/hex"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
)

type typeBSON struct {
	N string `bson:"name"`
	C string `bson:"code"`
}

func (ty Type) MarshalBSON() ([]byte, error) {
	name := ty.String()
	if len(name) < 1 {
		return nil, xerrors.Errorf("Type does not have name: %s", ty.Verbose())
	}

	return bson.Marshal(typeBSON{
		N: name,
		C: hex.EncodeToString(ty.Bytes()),
	})
}

func (ty *Type) UnmarshalBSON(b []byte) error {
	var o typeBSON
	if err := bson.Unmarshal(b, &o); err != nil {
		return err
	}

	return ty.unpack(o.C)
}
