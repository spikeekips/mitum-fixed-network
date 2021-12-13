package mongodbstorage

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/localtime"
	"go.mongodb.org/mongo-driver/bson"
)

type StagedOperation struct {
	BaseDoc
	op operation.Operation
}

func NewStagedOperation(op operation.Operation, enc encoder.Encoder) (StagedOperation, error) {
	b, err := NewBaseDoc(op.Fact().Hash().String(), op, enc)
	if err != nil {
		return StagedOperation{}, err
	}

	return StagedOperation{
		BaseDoc: b,
		op:      op,
	}, nil
}

func (sd StagedOperation) bsonM() (bson.M, error) {
	m, err := sd.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["hash"] = sd.op.Hash().String()
	m["inserted_at"] = localtime.UTCNow()

	return m, nil
}

func (sd StagedOperation) MarshalBSON() ([]byte, error) {
	m, err := sd.bsonM()
	if err != nil {
		return nil, err
	}

	return bsonenc.Marshal(m)
}

func loadStagedOperationFromDecoder(decoder func(interface{}) error, encs *encoder.Encoders) (operation.Operation, error) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return nil, err
	}

	_, hinter, err := LoadDataFromDoc(b, encs)
	if err != nil {
		return nil, err
	}

	op, ok := hinter.(operation.Operation)
	if !ok {
		return nil, errors.Errorf("not operation.Operation: %T", hinter)
	}

	return op, nil
}
