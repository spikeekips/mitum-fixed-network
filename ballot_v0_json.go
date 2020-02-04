package mitum

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/errors"
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
	valuehash.Hash, // seal hash
	valuehash.Hash, // body hash
	valuehash.Hash, // fact hash
	key.Signature, // fact signature
	BaseBallotV0,
	BaseBallotV0Fact,
	error,
) {
	var err error

	// signer
	var signer key.Publickey
	if signer, err = decodePublickey(enc, nib.SN); err != nil {
		return nil, nil, nil, nil, BaseBallotV0{}, BaseBallotV0Fact{}, err
	}

	var eh, ebh, efh valuehash.Hash
	if eh, err = decodeHash(enc, nib.H); err != nil {
		return nil, nil, nil, nil, BaseBallotV0{}, BaseBallotV0Fact{}, err
	}

	if ebh, err = decodeHash(enc, nib.BH); err != nil {
		return nil, nil, nil, nil, BaseBallotV0{}, BaseBallotV0Fact{}, err
	}

	if efh, err = decodeHash(enc, nib.FH); err != nil {
		return nil, nil, nil, nil, BaseBallotV0{}, BaseBallotV0Fact{}, err
	}

	var node Address
	if node, err = decodeAddress(enc, nib.N); err != nil {
		return nil, nil, nil, nil, BaseBallotV0{}, BaseBallotV0Fact{}, err
	}

	return eh, ebh, efh, nib.FSG,
		BaseBallotV0{
			signer:    signer,
			signature: nib.SG,
			signedAt:  nib.SA.Time,
			node:      node,
		},
		BaseBallotV0Fact{
			height: nib.HT,
			round:  nib.RD,
		}, nil
}

func decodeHash(enc *encoder.JSONEncoder, b []byte) (valuehash.Hash, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if v, ok := i.(valuehash.Hash); !ok {
		return nil, errors.InvalidTypeError.Wrapf("not valuehash.Hash; type=%T", i)
	} else {
		return v, nil
	}
}

func decodePublickey(enc *encoder.JSONEncoder, b []byte) (key.Publickey, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if v, ok := i.(key.Publickey); !ok {
		return nil, errors.InvalidTypeError.Wrapf("not key.Publickey; type=%T", i)
	} else {
		return v, nil
	}
}

func decodeAddress(enc *encoder.JSONEncoder, b []byte) (Address, error) {
	if i, err := enc.DecodeByHint(b); err != nil {
		return nil, err
	} else if v, ok := i.(Address); !ok {
		return nil, errors.InvalidTypeError.Wrapf("not Address; type=%T", i)
	} else {
		return v, nil
	}
}
