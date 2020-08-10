package mongodbstorage

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
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

	m["fact_hash_string"] = od.op.Fact().Hash().String()
	m["hash_string"] = od.op.Hash().String()
	m["hash"] = od.op.Hash()
	m["height"] = od.height

	return bsonenc.Marshal(m)
}
