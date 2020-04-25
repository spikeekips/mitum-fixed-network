package mongodbstorage

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/encoder"
	"go.mongodb.org/mongo-driver/bson"
)

type OperationDoc struct {
	BaseDoc
	op operation.Operation
}

func NewOperationDoc(op operation.Operation, enc encoder.Encoder) (OperationDoc, error) {
	b, err := NewBaseDoc(op.Hash().String(), op, enc)
	if err != nil {
		return OperationDoc{}, err
	}

	return OperationDoc{
		BaseDoc: b,
		op:      op,
	}, nil
}

func (od OperationDoc) MarshalBSON() ([]byte, error) {
	m, err := od.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["hash"] = od.op.Hash()

	return bson.Marshal(m)
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
