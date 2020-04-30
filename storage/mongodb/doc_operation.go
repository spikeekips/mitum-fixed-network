package mongodbstorage

import (
	"golang.org/x/xerrors"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/encoder"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
)

type OperationDoc struct {
	BaseDoc
	op     operation.Operation
	height base.Height
}

func NewOperationDoc(op operation.Operation, enc encoder.Encoder, height base.Height) (OperationDoc, error) {
	b, err := NewBaseDoc(nil, op, enc)
	if err != nil {
		return OperationDoc{}, err
	}

	return OperationDoc{
		BaseDoc: b,
		op:      op,
		height:  height,
	}, nil
}

func (od OperationDoc) MarshalBSON() ([]byte, error) {
	m, err := od.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["hash_string"] = od.op.Hash().String()
	m["hash"] = od.op.Hash()
	m["height"] = od.height

	return bsonencoder.Marshal(m)
}

func loadOperationFromDecoder(decoder func(interface{}) error, encs *encoder.Encoders) (
	operation.Operation, error,
) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return nil, err
	}

	var op operation.Operation

	_, hinter, err := loadWithEncoder(b, encs)
	if err != nil {
		return nil, err
	} else if i, ok := hinter.(operation.Operation); !ok {
		return nil, xerrors.Errorf("not Operation: %T", hinter)
	} else {
		op = i
	}

	return op, nil
}
