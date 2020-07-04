package ballot

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

func PackBaseBallotV0BSON(ballot Ballot) bson.M {
	return bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(ballot.Hint()),
		bson.M{
			"hash":           ballot.Hash(),
			"signer":         ballot.Signer(),
			"signature":      ballot.Signature(),
			"signed_at":      ballot.SignedAt(),
			"height":         ballot.Height(),
			"round":          ballot.Round(),
			"node":           ballot.Node(),
			"body_hash":      ballot.BodyHash(),
			"fact_signature": ballot.FactSignature(),
		},
	)
}

type BaseBallotV0UnpackerBSON struct {
	H   valuehash.Bytes `bson:"hash"`
	SN  key.KeyDecoder  `bson:"signer"`
	SG  key.Signature   `bson:"signature"`
	SA  time.Time       `bson:"signed_at"`
	HT  base.Height     `bson:"height"`
	RD  base.Round      `bson:"round"`
	N   bson.Raw        `bson:"node"`
	BH  valuehash.Bytes `bson:"body_hash"`
	FSG key.Signature   `bson:"fact_signature"`
}

func NewBaseBallotFactV0PackerBSON(bbf BaseBallotFactV0, ht hint.Hint) bson.M {
	return bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(ht),
		bson.M{
			"height": bbf.height,
			"round":  bbf.round,
		},
	)
}

type BaseBallotFactV0PackerXSON struct {
	HT base.Height `json:"height" bson:"height"`
	RD base.Round  `json:"round" bson:"round"`
}
