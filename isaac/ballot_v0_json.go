package isaac

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/valuehash"
)

type BaseBallotV0PackerJSON struct {
	encoder.JSONPackHintedHead
	H   valuehash.Hash     `json:"hash"`
	SN  key.Publickey      `json:"signer"`
	SG  key.Signature      `json:"signature"`
	SA  localtime.JSONTime `json:"signed_at"`
	HT  Height             `json:"height"`
	RD  Round              `json:"round"`
	N   Address            `json:"node"`
	BH  valuehash.Hash     `json:"body_hash"`
	FH  valuehash.Hash     `json:"fact_hash"`
	FSG key.Signature      `json:"fact_signature"`
}

func PackBaseBallotV0JSON(ballot Ballot) (BaseBallotV0PackerJSON, error) {
	return BaseBallotV0PackerJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(ballot.Hint()),
		H:                  ballot.Hash(),
		SN:                 ballot.Signer(),
		SG:                 ballot.Signature(),
		SA:                 localtime.NewJSONTime(ballot.SignedAt()),
		HT:                 ballot.Height(),
		RD:                 ballot.Round(),
		N:                  ballot.Node(),
		BH:                 ballot.BodyHash(),
		FH:                 ballot.FactHash(),
		FSG:                ballot.FactSignature(),
	}, nil
}

type BaseBallotV0UnpackerJSON struct {
	encoder.JSONPackHintedHead
	H   json.RawMessage    `json:"hash"`
	SN  json.RawMessage    `json:"signer"`
	SG  key.Signature      `json:"signature"`
	SA  localtime.JSONTime `json:"signed_at"`
	HT  Height             `json:"height"`
	RD  Round              `json:"round"`
	N   json.RawMessage    `json:"node"`
	BH  json.RawMessage    `json:"body_hash"`
	FH  json.RawMessage    `json:"fact_hash"`
	FSG key.Signature      `json:"fact_signature"`
}

func UnpackBaseBallotV0JSON(nib BaseBallotV0UnpackerJSON, enc *encoder.JSONEncoder) (
	valuehash.Hash, // body hash
	valuehash.Hash, // fact hash
	key.Signature, // fact signature
	BaseBallotV0,
	BaseBallotFactV0,
	error,
) {
	var err error

	// signer
	var signer key.Publickey
	if signer, err = decodePublickey(enc, nib.SN); err != nil {
		return nil, nil, nil, BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	var eh, ebh, efh valuehash.Hash
	if eh, err = decodeHash(enc, nib.H); err != nil {
		return nil, nil, nil, BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	if ebh, err = decodeHash(enc, nib.BH); err != nil {
		return nil, nil, nil, BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	if efh, err = decodeHash(enc, nib.FH); err != nil {
		return nil, nil, nil, BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	var node Address
	if node, err = decodeAddress(enc, nib.N); err != nil {
		return nil, nil, nil, BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	return ebh, efh, nib.FSG,
		BaseBallotV0{
			h:         eh,
			signer:    signer,
			signature: nib.SG,
			signedAt:  nib.SA.Time,
			node:      node,
		},
		BaseBallotFactV0{
			height: nib.HT,
			round:  nib.RD,
		}, nil
}

type BaseBallotFactV0PackerJSON struct {
	encoder.JSONPackHintedHead
	HT Height `json:"height"`
	RD Round  `json:"round"`
}

func NewBaseBallotFactV0PackerJSON(bbf BaseBallotFactV0, ht hint.Hint) BaseBallotFactV0PackerJSON {
	return BaseBallotFactV0PackerJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(ht),
		HT:                 bbf.height,
		RD:                 bbf.round,
	}
}
