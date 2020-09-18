package mongodbstorage

import (
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Doc interface {
	ID() interface{}
}

type BaseDoc struct {
	id          interface{}
	encoderHint hint.Hint
	data        interface{}
	isHinted    bool
}

func NewBaseDoc(id, data interface{}, enc encoder.Encoder) (BaseDoc, error) {
	_, isHinted := data.(hint.Hinter)

	return BaseDoc{
		id:          id,
		encoderHint: enc.Hint(),
		isHinted:    isHinted,
		data:        data,
	}, nil
}

func (do BaseDoc) ID() interface{} {
	return do.id
}

func (do BaseDoc) M() (bson.M, error) {
	m := bson.M{
		"_e":      do.encoderHint,
		"_hinted": do.isHinted,
		"d":       do.data,
	}

	if do.id != nil {
		m["_id"] = do.id
	}

	return m, nil
}

type BaseDocUnpacker struct {
	I bson.Raw      `bson:"_id,omitempty"`
	E hint.Hint     `bson:"_e"`
	D bson.RawValue `bson:"d"`
	H bool          `bson:"_hinted"`
}

func LoadDataFromDoc(b []byte, encs *encoder.Encoders) (bson.Raw /* id */, interface{} /* data */, error) {
	var bd BaseDocUnpacker
	if err := bsonenc.Unmarshal(b, &bd); err != nil {
		return nil, nil, err
	}

	enc, err := encs.Encoder(bd.E.Type(), bd.E.Version())
	if err != nil {
		return nil, nil, err
	}

	if !bd.H {
		return bd.I, bd.D, nil
	}

	var doc bson.Raw
	if d, ok := bd.D.DocumentOK(); !ok {
		return nil, nil, xerrors.Errorf("hinted should be mongodb Document")
	} else {
		doc = d
	}

	var data hint.Hinter
	if i, err := enc.DecodeByHint([]byte(doc)); err != nil {
		return nil, nil, err
	} else {
		data = i
	}

	return bd.I, data, nil
}

type HashIDDoc struct {
	I bson.Raw        `bson:"_id"`
	E hint.Hint       `bson:"_e"`
	H valuehash.Bytes `bson:"hash"`
}
