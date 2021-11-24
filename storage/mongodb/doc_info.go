package mongodbstorage

import (
	"github.com/btcsuite/btcutil/base58"
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/sha3"
)

const lastManifestDocID = "confirmed_block"

type lastManifestDoc struct {
	BaseDoc
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
		return base.NilHeight, err
	}

	var height base.Height
	_, d, err := LoadDataFromDoc(b, encs)
	if err != nil {
		return base.NilHeight, err
	} else if r, ok := d.(bson.RawValue); !ok {
		return base.NilHeight, errors.Errorf("invalid height: %T", d)
	} else if err := r.Unmarshal(&height); err != nil {
		return base.NilHeight, err
	}

	return height, nil
}

type InfoDoc struct {
	BaseDoc
	key string
}

func infoDocKey(key string) string {
	h := sha3.Sum256([]byte(key))

	return base58.Encode(h[:])
}

func NewInfoDoc(key string, v []byte, enc encoder.Encoder) (InfoDoc, error) {
	b, err := NewBaseDoc(infoDocKey(key), v, enc)
	if err != nil {
		return InfoDoc{}, err
	}

	return InfoDoc{BaseDoc: b, key: key}, nil
}

func (do InfoDoc) MarshalBSON() ([]byte, error) {
	m, err := do.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["key"] = do.key

	return bsonenc.Marshal(m)
}

func loadInfo(decoder func(interface{}) error, encs *encoder.Encoders) ([]byte /* value */, error) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return nil, err
	}

	var v []byte
	if _, d, err := LoadDataFromDoc(b, encs); err != nil {
		return nil, err
	} else if r, ok := d.(bson.RawValue); !ok {
		return nil, errors.Errorf("invalid data type for info, %T", d)
	} else if err := r.Unmarshal(&v); err != nil {
		return nil, err
	} else {
		return v, nil
	}
}
