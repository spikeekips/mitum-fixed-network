package mongodbstorage

import (
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BlockDoc struct {
	BaseDoc
	block block.Block
}

func NewBlockDoc(blk block.Block, enc encoder.Encoder) (BlockDoc, error) {
	b, err := NewBaseDoc(blk.Hash().String(), blk, enc)
	if err != nil {
		return BlockDoc{}, err
	}

	return BlockDoc{
		BaseDoc: b,
		block:   blk,
	}, nil
}

func (bd BlockDoc) MarshalBSON() ([]byte, error) {
	m, err := bd.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["height"] = bd.block.Height()
	m["hash"] = bd.block.Hash()

	return bsonenc.Marshal(m)
}

type ManifestDoc struct {
	BaseDoc
	manifest block.Manifest
}

func NewManifestDoc(manifest block.Manifest, enc encoder.Encoder) (ManifestDoc, error) {
	b, err := NewBaseDoc(manifest.Hash().String(), manifest, enc)
	if err != nil {
		return ManifestDoc{}, err
	}

	return ManifestDoc{
		BaseDoc:  b,
		manifest: manifest,
	}, nil
}

func (md ManifestDoc) MarshalBSON() ([]byte, error) {
	m, err := md.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["height"] = md.manifest.Height()
	m["hash"] = md.manifest.Hash()

	return bsonenc.Marshal(m)
}

func loadManifestFromDecoder(decoder func(interface{}) error, encs *encoder.Encoders) (block.Manifest, error) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return nil, err
	}

	var manifest block.Manifest

	_, hinter, err := LoadDataFromDoc(b, encs)
	if err != nil {
		return nil, err
	} else if i, ok := hinter.(block.Manifest); !ok {
		return nil, xerrors.Errorf("not Manifest: %T", hinter)
	} else {
		manifest = i
	}

	return manifest, nil
}

func loadManifestHeightAndHash(decoder func(interface{}) error, _ *encoder.Encoders) (base.Height, valuehash.Hash, error) {
	var hd struct {
		HT base.Height     `bson:"height"`
		H  valuehash.Bytes `bson:"hash"`
	}

	if err := decoder(&hd); err != nil {
		return base.NilHeight, nil, err
	} else if hd.H.Empty() {
		return base.NilHeight, nil, xerrors.Errorf("empty hash for ManifestDoc")
	}

	return hd.HT, hd.H, nil
}
