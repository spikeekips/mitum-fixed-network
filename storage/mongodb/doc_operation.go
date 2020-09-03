package mongodbstorage

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

type OperationDoc struct {
	BaseDoc
	fact   valuehash.Hash
	height base.Height
}

func NewOperationDoc(fact valuehash.Hash, enc encoder.Encoder, height base.Height) (OperationDoc, error) {
	b, err := NewBaseDoc(nil, nil, enc)
	if err != nil {
		return OperationDoc{}, err
	}

	return OperationDoc{
		BaseDoc: b,
		height:  height,
		fact:    fact,
	}, nil
}

func (od OperationDoc) MarshalBSON() ([]byte, error) {
	m, err := od.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["fact_hash_string"] = od.fact.String()
	m["fact_hash"] = od.fact
	m["height"] = od.height

	return bsonenc.Marshal(m)
}
