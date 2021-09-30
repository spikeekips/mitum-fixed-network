package network

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (conn NilConnInfo) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(bsonenc.NewHintedDoc(conn.Hint()), bson.M{
		"name": conn.s,
	}))
}

type NilConnInfoUnpackerBSON struct {
	S string `bson:"name"`
}

func (conn *NilConnInfo) UnmarshalBSON(b []byte) error {
	var uc NilConnInfoUnpackerBSON
	if err := bsonenc.Unmarshal(b, &uc); err != nil {
		return err
	}

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
	U string `bson:"url"`
	I bool   `bson:"insecure"`
}

func (conn *HTTPConnInfo) UnmarshalBSON(b []byte) error {
	var uc HTTPConnInfoUnpackerBSON
	if err := bson.Unmarshal(b, &uc); err != nil {
		return err
	}

	return conn.unpack(uc.U, uc.I)
}
