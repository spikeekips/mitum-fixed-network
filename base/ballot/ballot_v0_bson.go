package ballot

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
)

func PackBaseBallotV0BSON(ballot Ballot) bson.M {
	return bsonencoder.MergeBSONM(
		bsonencoder.NewHintedDoc(ballot.Hint()),
		bson.M{
			"hash":           ballot.Hash(),
			"signer":         ballot.Signer(),
			"signature":      ballot.Signature(),
			"signed_at":      ballot.SignedAt(),
			"height":         ballot.Height(),
			"round":          ballot.Round(),
			"node":           ballot.Node(),
			"body_hash":      ballot.BodyHash(),
			"fact_hash":      ballot.FactHash(),
			"fact_signature": ballot.FactSignature(),
		},
	)
}

type BaseBallotV0UnpackerBSON struct {
	H   bson.Raw      `bson:"hash"`
	SN  bson.Raw      `bson:"signer"`
	SG  key.Signature `bson:"signature"`
	SA  time.Time     `bson:"signed_at"`
	HT  base.Height   `bson:"height"`
	RD  base.Round    `bson:"round"`
	N   bson.Raw      `bson:"node"`
	BH  bson.Raw      `bson:"body_hash"`
	FH  bson.Raw      `bson:"fact_hash"`
	FSG key.Signature `bson:"fact_signature"`
}

func NewBaseBallotFactV0PackerBSON(bbf BaseBallotFactV0, ht hint.Hint) bson.M {
	return bsonencoder.MergeBSONM(
		bsonencoder.NewHintedDoc(ht),
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
