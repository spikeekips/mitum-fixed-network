package encoder

import (
	"github.com/spikeekips/mitum/util/hint"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"
)

func (hs *HintedString) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	switch t {
	case bsontype.Null:
		return nil
	case bsontype.String:
	default:
		return xerrors.Errorf("invalid marshaled type for HintedString, %v", t)
	}

	s, _, ok := bsoncore.ReadString(b)
	if !ok {
		return xerrors.Errorf("can not read string")
	}

	h, us, err := hint.ParseHintedString(s)
	if err != nil {
		return err
	}
	hs.h = h
	hs.s = us

	return nil
}
