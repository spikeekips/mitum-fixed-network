package mitum

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
)

type BaseBallotV0Fact struct {
	height Height
	round  Round
}

func (bf BaseBallotV0Fact) IsReadyToSign(b []byte) error {
	if err := bf.height.IsValid(b); err != nil {
		return err
	}

	return nil
}

func (bf BaseBallotV0Fact) IsValid(b []byte) error {
	if err := bf.IsReadyToSign(b); err != nil {
		return err
	}

	return nil
}

func (bf BaseBallotV0Fact) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		bf.height.Bytes(),
		bf.round.Bytes(),
	})
}

func (bf BaseBallotV0Fact) Height() Height {
	return bf.height
}

func (bf BaseBallotV0Fact) Round() Round {
	return bf.round
}

type BaseBallotV0 struct {
	signer    key.Publickey
	signature key.Signature
	signedAt  time.Time
	node      Address
}

func NewBaseBallotV0(node Address) BaseBallotV0 {
	return BaseBallotV0{
		node: node,
	}
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

func (bb BaseBallotV0) Node() Address {
	return bb.node
}

func (bb BaseBallotV0) IsValid(b []byte) error {
	if err := bb.IsReadyToSign(b); err != nil {
		return err
	}

	if bb.signedAt.IsZero() {
		return xerrors.Errorf("empty SignedAt")
	}

	if bb.signer == nil {
		return xerrors.Errorf("empty Signer")
	} else if err := bb.signer.IsValid(b); err != nil {
		return err
	}

	if bb.signature == nil {
		return xerrors.Errorf("empty Signature")
	} else if err := bb.signature.IsValid(b); err != nil {
		return err
	}

	return nil
}

func (bb BaseBallotV0) IsReadyToSign(b []byte) error {
	if err := bb.node.IsValid(b); err != nil {
		return err
	}

	return nil
}

func (bb BaseBallotV0) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		[]byte(bb.signer.String()),
		bb.signature.Bytes(),
		[]byte(localtime.RFC3339(bb.signedAt)),
		bb.node.Bytes(),
	})
}
