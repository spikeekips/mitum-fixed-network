package encoder

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/hint"
)

func (hs *HintedString) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	if t != bsontype.String {
		return xerrors.Errorf("invalid marshaled type for HintedString, %v", t)
	}

	s, _, ok := bsoncore.ReadString(b)
	if !ok {
		return xerrors.Errorf("can not read string")
	}

	if h, us, err := hint.ParseHintedString(s); err != nil {
		return err
	} else {
		hs.h = h
		hs.s = us
	}

	return nil
}
