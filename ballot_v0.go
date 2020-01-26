package mitum

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
)

type BaseBallotV0 struct {
	signer    key.Publickey
	signature key.Signature
	signedAt  time.Time
	//-x-------------------- hashing parts
	height Height
	round  Round
	node   Address
	//--------------------x-
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

func (bb BaseBallotV0) Height() Height {
	return bb.height
}

func (bb BaseBallotV0) Round() Round {
	return bb.round
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
	if err := isvalid.Check([]isvalid.IsValider{bb.height, bb.node}, b); err != nil {
		return err
	}

	return nil
}

func (bb BaseBallotV0) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		[]byte(bb.signer.String()),
		bb.signature.Bytes(),
		[]byte(localtime.RFC3339(bb.signedAt)),
		bb.height.Bytes(),
		bb.round.Bytes(),
		bb.node.Bytes(),
	})
}

func (bb BaseBallotV0) BodyBytes() []byte {
	return util.ConcatSlice([][]byte{
		bb.height.Bytes(),
		bb.round.Bytes(),
		bb.node.Bytes(),
	})
}
