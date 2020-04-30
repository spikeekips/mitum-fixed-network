package mongodbstorage

import (
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util/encoder"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
)

const confirmedBlockDocID = "confirmed_block"

type ConfirmedBlockDoc struct {
	BaseDoc
	block block.Block
}

func NewConfirmedBlockDoc(height base.Height, enc encoder.Encoder) (ConfirmedBlockDoc, error) {
	b, err := NewBaseDoc(confirmedBlockDocID, height, enc)
	if err != nil {
		return ConfirmedBlockDoc{}, err
	}

	return ConfirmedBlockDoc{BaseDoc: b}, nil
}

func (bd ConfirmedBlockDoc) MarshalBSON() ([]byte, error) {
	m, err := bd.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	return bsonencoder.Marshal(m)
}

func loadConfirmedBlock(decoder func(interface{}) error, encs *encoder.Encoders) (base.Height, error) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return base.Height(0), err
	}

	var height base.Height
	_, d, err := loadWithEncoder(b, encs)
	if err != nil {
		return base.Height(0), err
	} else if r, ok := d.(bson.Raw); !ok {
		return base.Height(0), xerrors.Errorf("invalid height: %T", d)
	} else if err := bsonencoder.Unmarshal(r, &height); err != nil {
		return base.Height(0), err
	}

	return height, nil
}
