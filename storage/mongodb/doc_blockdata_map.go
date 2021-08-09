package mongodbstorage

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

type BlockDataMapDoc struct {
	BaseDoc
	bd block.BlockDataMap
}

func NewBlockDataMapDoc(bd block.BlockDataMap, enc encoder.Encoder) (BlockDataMapDoc, error) {
	b, err := NewBaseDoc(nil, bd, enc)
	if err != nil {
		return BlockDataMapDoc{}, err
	}

	return BlockDataMapDoc{
		BaseDoc: b,
		bd:      bd,
	}, nil
}

func (bd BlockDataMapDoc) MarshalBSON() ([]byte, error) {
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

func loadBlockDataMapFromDecoder(decoder func(interface{}) error, encs *encoder.Encoders) (block.BlockDataMap, error) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return nil, err
	}

	var bd block.BlockDataMap

	_, hinter, err := LoadDataFromDoc(b, encs)
	if err != nil {
		return nil, err
	} else if i, ok := hinter.(block.BlockDataMap); !ok {
		return nil, errors.Errorf("not block.BlockDataMap: %T", hinter)
	} else {
		bd = i
	}

	return bd, nil
}
