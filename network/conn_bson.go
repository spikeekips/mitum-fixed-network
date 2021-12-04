package network

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
)

func (conn NilConnInfo) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(bsonenc.NewHintedDoc(conn.Hint()), bson.M{
		"name": conn.s,
	}))
}

type NilConnInfoUnpackerBSON struct {
	HT hint.Hint `bson:"_hint"`
	S  string    `bson:"name"`
}

func (conn *NilConnInfo) UnmarshalBSON(b []byte) error {
	var uc NilConnInfoUnpackerBSON
	if err := bsonenc.Unmarshal(b, &uc); err != nil {
		return err
	}

	conn.BaseHinter = hint.NewBaseHinter(uc.HT)
	conn.s = uc.S

	return nil
}

func (conn HTTPConnInfo) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(bsonenc.NewHintedDoc(conn.Hint()), bson.M{
		"url":      conn.u.String(),
		"insecure": conn.insecure,
	}))
}

type HTTPConnInfoUnpackerBSON struct {
	HT hint.Hint `bson:"_hint"`
	U  string    `bson:"url"`
	I  bool      `bson:"insecure"`
}

func (conn *HTTPConnInfo) UnmarshalBSON(b []byte) error {
	var uc HTTPConnInfoUnpackerBSON
	if err := bson.Unmarshal(b, &uc); err != nil {
		return err
	}

	return conn.unpack(uc.HT, uc.U, uc.I)
}
