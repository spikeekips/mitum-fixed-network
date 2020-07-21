package ballot

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BaseBallotFactV0 struct {
	height base.Height
	round  base.Round
}

func NewBaseBallotFactV0(height base.Height, round base.Round) BaseBallotFactV0 {
	return BaseBallotFactV0{
		height: height,
		round:  round,
	}
}

func (bf BaseBallotFactV0) IsReadyToSign([]byte) error {
	if err := bf.height.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (bf BaseBallotFactV0) IsValid(networkID []byte) error {
	return bf.IsReadyToSign(networkID)
}

func (bf BaseBallotFactV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		bf.height.Bytes(),
		bf.round.Bytes(),
	)
}

func (bf BaseBallotFactV0) Height() base.Height {
	return bf.height
}

func (bf BaseBallotFactV0) Round() base.Round {
	return bf.round
}

type BaseBallotV0 struct {
	h             valuehash.Hash
	bodyHash      valuehash.Hash
	signer        key.Publickey
	signature     key.Signature
	signedAt      time.Time
	node          base.Address
	factSignature key.Signature
}

func NewBaseBallotV0(node base.Address) BaseBallotV0 {
	return BaseBallotV0{
		node: node,
	}
}

func (bb BaseBallotV0) Hash() valuehash.Hash {
	return bb.h
}

func (bb BaseBallotV0) SetHash(h valuehash.Hash) BaseBallotV0 {
	bb.h = h

	return bb
}

func (bb BaseBallotV0) BodyHash() valuehash.Hash {
	return bb.bodyHash
}

func (bb BaseBallotV0) Signer() key.Publickey {
	return bb.signer
}

func (bb BaseBallotV0) Signature() key.Signature {
	return bb.signature
}

func (bb BaseBallotV0) SignedAt() time.Time {
	return bb.signedAt
}

func (bb BaseBallotV0) FactSignature() key.Signature {
	return bb.factSignature
}

func (bb BaseBallotV0) Node() base.Address {
	return bb.node
}

func (bb BaseBallotV0) IsValid(networkID []byte) error {
	if err := bb.IsReadyToSign(networkID); err != nil {
		return err
	}

	if bb.signedAt.IsZero() {
		return xerrors.Errorf("empty SignedAt")
	}

	if bb.signer == nil {
		return xerrors.Errorf("empty Signer")
	} else if err := bb.signer.IsValid(nil); err != nil {
		return err
	}

	if bb.signature == nil {
		return xerrors.Errorf("empty Signature")
	} else if err := bb.signature.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (bb BaseBallotV0) IsReadyToSign([]byte) error {
	if err := bb.node.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (bb BaseBallotV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		bb.bodyHash.Bytes(),
		bb.signer.Bytes(),
		bb.signature.Bytes(),
		[]byte(localtime.String(localtime.Normalize(bb.signedAt))),
		bb.node.Bytes(),
		bb.factSignature.Bytes(),
	)
}
