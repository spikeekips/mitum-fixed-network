package block

import (
	"time"

	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (bd BaseBlockdataMap) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(bd.Hint()),
		bson.M{
			"hash":       bd.h,
			"height":     bd.height,
			"block":      bd.block,
			"created_at": bd.createdAt,
			"items":      bd.items,
			"writer":     bd.writerHint,
		},
	))
}

type BaseBlockdataMapBSONUnpacker struct {
	H         valuehash.Bytes     `bson:"hash"`
	Height    base.Height         `bson:"height"`
	Block     valuehash.Bytes     `bson:"block"`
	CreatedAt time.Time           `bson:"created_at"`
	Items     map[string]bson.Raw `bson:"items"`
	Writer    hint.Hint           `bson:"writer"`
}

func (bd *BaseBlockdataMap) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubd BaseBlockdataMapBSONUnpacker
	if err := enc.Unmarshal(b, &ubd); err != nil {
		return err
	}

	bitems := map[string][]byte{}
	for k := range ubd.Items {
		bitems[k] = ubd.Items[k]
	}

	return bd.unpack(enc, ubd.H, ubd.Height, ubd.Block, ubd.CreatedAt, bitems, ubd.Writer)
}

type BaseBlockdataMapItemBSONPacker struct {
	Type     string `bson:"type"`
	Checksum string `bson:"checksum"`
	URL      string `bson:"url"`
}

func (bd BaseBlockdataMapItem) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(BaseBlockdataMapItemBSONPacker{
		Type:     bd.t,
		Checksum: bd.checksum,
		URL:      bd.url,
	})
}

func (bd *BaseBlockdataMapItem) UnmarshalBSON(b []byte) error {
	var ubdi BaseBlockdataMapItemBSONPacker
	if err := bsonenc.Unmarshal(b, &ubdi); err != nil {
		return err
	}

	return bd.unpack(ubdi.Type, ubdi.Checksum, ubdi.URL)
}
