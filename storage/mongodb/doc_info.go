package mongodbstorage

import (
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

const lastManifestDocID = "confirmed_block"

type lastManifestDoc struct {
	BaseDoc
	block block.Block
}

func NewLastManifestDoc(height base.Height, enc encoder.Encoder) (lastManifestDoc, error) {
	b, err := NewBaseDoc(lastManifestDocID, height, enc)
	if err != nil {
		return lastManifestDoc{}, err
	}

	return lastManifestDoc{BaseDoc: b}, nil
}

func (bd lastManifestDoc) MarshalBSON() ([]byte, error) {
	m, err := bd.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	return bsonenc.Marshal(m)
}

func loadLastManifest(decoder func(interface{}) error, encs *encoder.Encoders) (base.Height, error) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return base.Height(0), err
	}

	var height base.Height
	_, d, err := loadWithEncoder(b, encs)
	if err != nil {
		return base.Height(0), err
	} else if r, ok := d.(bson.RawValue); !ok {
		return base.Height(0), xerrors.Errorf("invalid height: %T", d)
	} else if err := r.Unmarshal(&height); err != nil {
		return base.Height(0), err
	}

	return height, nil
}
