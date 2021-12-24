package mongodbstorage

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

type BlockdataMapDoc struct {
	BaseDoc
	bd block.BlockdataMap
}

func NewBlockdataMapDoc(bd block.BlockdataMap, enc encoder.Encoder) (BlockdataMapDoc, error) {
	b, err := NewBaseDoc(nil, bd, enc)
	if err != nil {
		return BlockdataMapDoc{}, err
	}

	return BlockdataMapDoc{
		BaseDoc: b,
		bd:      bd,
	}, nil
}

func (bd BlockdataMapDoc) MarshalBSON() ([]byte, error) {
	m, err := bd.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["height"] = bd.bd.Height()
	m["hash"] = bd.bd.Hash()
	m["block_hash"] = bd.bd.Block()
	m["is_local"] = bd.bd.IsLocal()

	return bsonenc.Marshal(m)
}

func loadBlockdataMapFromDecoder(decoder func(interface{}) error, encs *encoder.Encoders) (block.BlockdataMap, error) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return nil, err
	}

	var bd block.BlockdataMap

	_, hinter, err := LoadDataFromDoc(b, encs)
	if err != nil {
		return nil, err
	} else if i, ok := hinter.(block.BlockdataMap); !ok {
		return nil, errors.Errorf("not block.BlockdataMap: %T", hinter)
	} else {
		bd = i
	}

	return bd, nil
}
