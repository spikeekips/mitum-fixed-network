package mongodbstorage

import (
	"golang.org/x/xerrors"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/encoder"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
)

type StateDoc struct {
	BaseDoc
	state state.State
}

func NewStateDoc(st state.State, enc encoder.Encoder) (StateDoc, error) {
	b, err := NewBaseDoc(nil, st, enc)
	if err != nil {
		return StateDoc{}, err
	}

	return StateDoc{
		BaseDoc: b,
		state:   st,
	}, nil
}

func (sd StateDoc) MarshalBSON() ([]byte, error) {
	m, err := sd.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["key"] = sd.state.Key()
	m["height"] = sd.state.Height()

	return bsonencoder.Marshal(m)
}

func loadStateFromDecoder(decoder func(interface{}) error, encs *encoder.Encoders) (
	state.State, error,
) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return nil, err
	}

	var st state.State

	_, hinter, err := loadWithEncoder(b, encs)
	if err != nil {
		return nil, err
	} else if i, ok := hinter.(state.State); !ok {
		return nil, xerrors.Errorf("not state.State: %T", hinter)
	} else {
		st = i
	}

	return st, nil
}
