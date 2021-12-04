package base

import (
	"fmt"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

func (fs BaseBallotFactSign) MarshalBSON() ([]byte, error) {
	b, err := bsonenc.Marshal(fs.BaseFactSign)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ballot fact sign: %w", err)
	}

	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(fs.Hint()),
		bson.M{
			"base": bson.Raw(b),
			"node": fs.node,
		},
	))
}

type BaseBallotFactSignNodeBSONUnpacker struct {
	B  bson.Raw       `bson:"base"`
	NO AddressDecoder `bson:"node"`
}

func (fs *BaseBallotFactSign) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var bn BaseBallotFactSignNodeBSONUnpacker
	if err := enc.Unmarshal(b, &bn); err != nil {
		return fmt.Errorf("failed to unpack ballot fact sign: %w", err)
	}

	var bfs BaseFactSign
	if err := bfs.UnpackBSON(bn.B, enc); err != nil {
		return fmt.Errorf("failed to unpack ballot factsign: %w", err)
	}

	return fs.unpack(enc, bfs, bn.NO)
}

func (sfs BaseSignedBallotFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(bsonenc.NewHintedDoc(sfs.Hint()), bson.M{
		"fact":      sfs.fact,
		"fact_sign": sfs.factSign,
	}))
}

type BaseSignedBallotFactUnpackBSON struct {
	FC bson.Raw `bson:"fact"`
	FS bson.Raw `bson:"fact_sign"`
}

func (sfs *BaseSignedBallotFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var vpp BaseSignedBallotFactUnpackBSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	return sfs.unpack(enc, vpp.FC, vpp.FS)
}
