package ballot

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (BaseBallotV0) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	bSigner key.PublickeyDecoder,
	signature key.Signature,
	signedAt time.Time,
	height base.Height,
	round base.Round,
	bNode base.AddressDecoder,
	bodyHash valuehash.Hash,
	factSignature key.Signature,
) (BaseBallotV0, BaseBallotFactV0, error) {
	// signer
	signer, err := bSigner.Encode(enc)
	if err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	node, err := bNode.Encode(enc)
	if err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	return BaseBallotV0{
			h:             h,
			bodyHash:      bodyHash,
			signer:        signer,
			signature:     signature,
			signedAt:      signedAt,
			node:          node,
			factSignature: factSignature,
		},
		BaseBallotFactV0{
			height: height,
			round:  round,
		}, nil
}

func (bb BaseBallotV0) unpackJSON(b []byte, enc *jsonenc.Encoder) (
	BaseBallotV0, BaseBallotFactV0, error,
) {
	var nbb BaseBallotV0UnpackerJSON
	if err := jsonenc.Unmarshal(b, &nbb); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	return bb.unpack(enc,
		nbb.H, nbb.SN, nbb.SG, nbb.SA.Time, nbb.HT, nbb.RD, nbb.N, nbb.BH, nbb.FSG)
}

func (bb BaseBallotV0) unpackBSON(b []byte, enc *bsonenc.Encoder) (
	BaseBallotV0, BaseBallotFactV0, error,
) {
	var nbb BaseBallotV0UnpackerBSON
	if err := bsonenc.Unmarshal(b, &nbb); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	return bb.unpack(enc,
		nbb.H, nbb.SN, nbb.SG, nbb.SA, nbb.HT, nbb.RD, nbb.N, nbb.BH, nbb.FSG)
}

func (BaseBallotFactV0) unpack(_ encoder.Encoder, height base.Height, round base.Round) (
	BaseBallotFactV0, error,
) {
	return NewBaseBallotFactV0(height, round), nil
}

func (bf BaseBallotFactV0) unpackJSON(b []byte, enc *jsonenc.Encoder) (BaseBallotFactV0, error) {
	var ubbf BaseBallotFactV0PackerXSON
	if err := enc.Unmarshal(b, &ubbf); err != nil {
		return BaseBallotFactV0{}, err
	}

	return bf.unpack(enc, ubbf.HT, ubbf.RD)
}

func (bf BaseBallotFactV0) unpackBSON(b []byte, enc *bsonenc.Encoder) (BaseBallotFactV0, error) {
	var ubbf BaseBallotFactV0PackerXSON
	if err := enc.Unmarshal(b, &ubbf); err != nil {
		return BaseBallotFactV0{}, err
	}

	return bf.unpack(enc, ubbf.HT, ubbf.RD)
}
