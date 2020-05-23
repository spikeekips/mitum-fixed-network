package ballot

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

func (bb BaseBallotV0) unpack(
	enc encoder.Encoder,
	bHash,
	bSigner []byte,
	signature key.Signature,
	signedAt time.Time,
	height base.Height,
	round base.Round,
	bNode,
	bBodyHash,
	bFactHash []byte,
	factSignature key.Signature,
) (BaseBallotV0, BaseBallotFactV0, error) {
	var err error

	// signer
	var signer key.Publickey
	if signer, err = key.DecodePublickey(enc, bSigner); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	var eh, ebh, efh valuehash.Hash
	if eh, err = valuehash.Decode(enc, bHash); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	if ebh, err = valuehash.Decode(enc, bBodyHash); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	if efh, err = valuehash.Decode(enc, bFactHash); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	var node base.Address
	if node, err = base.DecodeAddress(enc, bNode); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	return BaseBallotV0{
			h:             eh,
			bodyHash:      ebh,
			signer:        signer,
			signature:     signature,
			signedAt:      signedAt,
			node:          node,
			factHash:      efh,
			factSignature: factSignature,
		},
		BaseBallotFactV0{
			height: height,
			round:  round,
		}, nil
}

func (bb BaseBallotV0) unpackJSON(b []byte, enc *jsonencoder.Encoder) (
	BaseBallotV0, BaseBallotFactV0, error,
) {
	var nbb BaseBallotV0UnpackerJSON
	if err := jsonencoder.Unmarshal(b, &nbb); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	return bb.unpack(enc,
		nbb.H, nbb.SN, nbb.SG, nbb.SA.Time, nbb.HT, nbb.RD, nbb.N, nbb.BH, nbb.FH, nbb.FSG)
}

func (bb BaseBallotV0) unpackBSON(b []byte, enc *bsonencoder.Encoder) (
	BaseBallotV0, BaseBallotFactV0, error,
) {
	var nbb BaseBallotV0UnpackerBSON
	if err := bsonencoder.Unmarshal(b, &nbb); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	return bb.unpack(enc,
		nbb.H, nbb.SN, nbb.SG, nbb.SA, nbb.HT, nbb.RD, nbb.N, nbb.BH, nbb.FH, nbb.FSG)
}

func (bf BaseBallotFactV0) unpack(_ encoder.Encoder, height base.Height, round base.Round) (
	BaseBallotFactV0, error,
) {
	return NewBaseBallotFactV0(height, round), nil
}

func (bf BaseBallotFactV0) unpackJSON(b []byte, enc *jsonencoder.Encoder) (BaseBallotFactV0, error) {
	var ubbf BaseBallotFactV0PackerXSON
	if err := enc.Unmarshal(b, &ubbf); err != nil {
		return BaseBallotFactV0{}, err
	}

	return bf.unpack(enc, ubbf.HT, ubbf.RD)
}

func (bf BaseBallotFactV0) unpackBSON(b []byte, enc *bsonencoder.Encoder) (BaseBallotFactV0, error) {
	var ubbf BaseBallotFactV0PackerXSON
	if err := enc.Unmarshal(b, &ubbf); err != nil {
		return BaseBallotFactV0{}, err
	}

	return bf.unpack(enc, ubbf.HT, ubbf.RD)
}
