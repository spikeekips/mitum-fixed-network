package operation

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/key"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/localtime"
)

func (fs BaseFactSign) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(fs.Hint()),
		bson.M{
			"signer":    fs.signer,
			"signature": fs.signature,
			"signed_at": localtime.NewJSONTime(fs.signedAt),
		},
	))
}

type BaseFactSignBSONUnpacker struct {
	SN key.KeyDecoder     `bson:"signer"`
	SG key.Signature      `bson:"signature"`
	SA localtime.JSONTime `bson:"signed_at"`
}

func (fs *BaseFactSign) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ufs BaseFactSignBSONUnpacker
	if err := enc.Unmarshal(b, &ufs); err != nil {
		return err
	}

	return fs.unpack(enc, ufs.SN, ufs.SG, ufs.SA.Time)
}
