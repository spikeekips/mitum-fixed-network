package isaac

import (
	"encoding/json"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
)

type BaseBallotV0PackerJSON struct {
	encoder.JSONPackHintedHead
	H   valuehash.Hash     `json:"hash"`
	SN  key.Publickey      `json:"signer"`
	SG  key.Signature      `json:"signature"`
	SA  localtime.JSONTime `json:"signed_at"`
	HT  base.Height        `json:"height"`
	RD  base.Round         `json:"round"`
	N   base.Address       `json:"node"`
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
	HT  base.Height        `json:"height"`
	RD  base.Round         `json:"round"`
	N   json.RawMessage    `json:"node"`
	BH  json.RawMessage    `json:"body_hash"`
	FH  json.RawMessage    `json:"fact_hash"`
	FSG key.Signature      `json:"fact_signature"`
}

func UnpackBaseBallotV0JSON(nib BaseBallotV0UnpackerJSON, enc *encoder.JSONEncoder) (
	BaseBallotV0,
	BaseBallotFactV0,
	error,
) {
	var err error

	// signer
	var signer key.Publickey
	if signer, err = key.DecodePublickey(enc, nib.SN); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	var eh, ebh, efh valuehash.Hash
	if eh, err = valuehash.Decode(enc, nib.H); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	if ebh, err = valuehash.Decode(enc, nib.BH); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	if efh, err = valuehash.Decode(enc, nib.FH); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	var node base.Address
	if node, err = base.DecodeAddress(enc, nib.N); err != nil {
		return BaseBallotV0{}, BaseBallotFactV0{}, err
	}

	return BaseBallotV0{
			h:             eh,
			bodyHash:      ebh,
			signer:        signer,
			signature:     nib.SG,
			signedAt:      nib.SA.Time,
			node:          node,
			factHash:      efh,
			factSignature: nib.FSG,
		},
		BaseBallotFactV0{
			height: nib.HT,
			round:  nib.RD,
		}, nil
}

type BaseBallotFactV0PackerJSON struct {
	encoder.JSONPackHintedHead
	HT base.Height `json:"height"`
	RD base.Round  `json:"round"`
}

func NewBaseBallotFactV0PackerJSON(bbf BaseBallotFactV0, ht hint.Hint) BaseBallotFactV0PackerJSON {
	return BaseBallotFactV0PackerJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(ht),
		HT:                 bbf.height,
		RD:                 bbf.round,
	}
}
