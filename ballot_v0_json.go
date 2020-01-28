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
	H  json.RawMessage    `json:"hash"`
	SN json.RawMessage    `json:"signer"`
	SG key.Signature      `json:"signature"`
	SA localtime.JSONTime `json:"signed_at"`
	HT Height             `json:"height"`
	RD Round              `json:"round"`
	N  json.RawMessage    `json:"node"`
	BH json.RawMessage    `json:"fact_hash"`
}

func PackBaseBallotJSON(ballot Ballot, enc *encoder.JSONEncoder) (BaseBallotV0PackerJSON, error) {
	var jh, jbh, ja json.RawMessage
	if h, err := enc.Marshal(ballot.Hash()); err != nil {
		return BaseBallotV0PackerJSON{}, err
	} else {
		jh = h
	}
	if h, err := enc.Marshal(ballot.BodyHash()); err != nil {
		return BaseBallotV0PackerJSON{}, err
	} else {
		jbh = h
	}
	if h, err := enc.Encode(ballot.Node()); err != nil {
		return BaseBallotV0PackerJSON{}, err
	} else {
		ja = h
	}

	bs, err := enc.Encode(ballot.Signer())
	if err != nil {
		return BaseBallotV0PackerJSON{}, err
	}

	return BaseBallotV0PackerJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(ballot.Hint()),
		H:                  jh,
		SN:                 bs,
		SG:                 ballot.Signature(),
		SA:                 localtime.NewJSONTime(ballot.SignedAt()),
		HT:                 ballot.Height(),
		RD:                 ballot.Round(),
		N:                  ja,
		BH:                 jbh,
	}, nil
}

func UnpackBaseBallotJSON(nib BaseBallotV0PackerJSON, enc *encoder.JSONEncoder) (
	valuehash.Hash, valuehash.Hash, BaseBallotV0, BaseBallotV0Fact, error,
) {
	// signer
	var signer key.Publickey
	if i, err := enc.DecodeByHint(nib.SN); err != nil {
		return nil, nil, BaseBallotV0{}, BaseBallotV0Fact{}, err
	} else if v, ok := i.(key.Publickey); !ok {
		return nil, nil, BaseBallotV0{}, BaseBallotV0Fact{}, errors.InvalidTypeError.Wrapf("not key.Publickey; type=%T", i)
	} else {
		signer = v
	}

	var eh, ebh valuehash.Hash

	// seal hash
	if i, err := enc.DecodeByHint(nib.H); err != nil {
		return nil, nil, BaseBallotV0{}, BaseBallotV0Fact{}, err
	} else if v, ok := i.(valuehash.Hash); !ok {
		return nil, nil, BaseBallotV0{}, BaseBallotV0Fact{}, errors.InvalidTypeError.Wrapf("not valuehash.Hash; type=%T", i)
	} else {
		eh = v
	}

	// bodyhash
	if i, err := enc.DecodeByHint(nib.BH); err != nil {
		return nil, nil, BaseBallotV0{}, BaseBallotV0Fact{}, err
	} else if v, ok := i.(valuehash.Hash); !ok {
		return nil, nil, BaseBallotV0{}, BaseBallotV0Fact{}, errors.InvalidTypeError.Wrapf("not valuehash.Hash; type=%T", i)
	} else {
		ebh = v
	}

	var node Address
	if i, err := enc.DecodeByHint(nib.N); err != nil {
		return nil, nil, BaseBallotV0{}, BaseBallotV0Fact{}, err
	} else if v, ok := i.(Address); !ok {
		return nil, nil, BaseBallotV0{}, BaseBallotV0Fact{}, errors.InvalidTypeError.Wrapf("not Address; type=%T", i)
	} else {
		node = v
	}

	return eh,
		ebh,
		BaseBallotV0{
			signer:    signer,
			signature: nib.SG,
			signedAt:  nib.SA.Time,
			node:      node,
		},
		BaseBallotV0Fact{
			height: nib.HT,
			round:  nib.RD,
		},
		nil
}
