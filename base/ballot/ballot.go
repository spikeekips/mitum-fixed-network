package ballot

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	INITBallotType         = hint.MustNewType(0x01, 0x20, "init-ballot")
	ProposalBallotType     = hint.MustNewType(0x01, 0x21, "proposal")
	SIGNBallotType         = hint.MustNewType(0x01, 0x22, "sign-ballot")
	ACCEPTBallotType       = hint.MustNewType(0x01, 0x23, "accept-ballot")
	INITBallotFactType     = hint.MustNewType(0x01, 0x24, "init-ballot-fact")
	ProposalBallotFactType = hint.MustNewType(0x01, 0x25, "proposal-fact")
	SIGNBallotFactType     = hint.MustNewType(0x01, 0x26, "sign-ballot-fact")
	ACCEPTBallotFactType   = hint.MustNewType(0x01, 0x27, "accept-ballot-fact")
)

type Ballot interface {
	seal.Seal
	Fact() base.Fact
	FactSignature() key.Signature
	logging.LogHintedMarshaler
	Stage() base.Stage
	Height() base.Height
	Round() base.Round
	Node() base.Address
}

type INITBallot interface {
	Ballot
	PreviousBlock() valuehash.Hash
	Voteproof() base.Voteproof
}

type Proposal interface {
	Ballot
	Seals() []valuehash.Hash // NOTE collection of received Seals, which must have Operations()
}

type SIGNBallot interface {
	Ballot
	Proposal() valuehash.Hash
	NewBlock() valuehash.Hash
}

type ACCEPTBallot interface {
	Ballot
	Proposal() valuehash.Hash
	NewBlock() valuehash.Hash
	Voteproof() base.Voteproof
}

type INITBallotFact interface {
	valuehash.Hasher
	PreviousBlock() valuehash.Hash
}

type SIGNBallotFact interface {
	valuehash.Hasher
	Proposal() valuehash.Hash
	NewBlock() valuehash.Hash
}

type ACCEPTBallotFact interface {
	valuehash.Hasher
	Proposal() valuehash.Hash
	NewBlock() valuehash.Hash
}

func IsValidBallot(blt Ballot, b []byte) error {
	if err := seal.IsValidSeal(blt, b); err != nil {
		return err
	}

	if blt.Fact() == nil {
		return isvalid.InvalidError.Errorf("Ballot has empty Fact()")
	}
	if blt.Fact().Hash() == nil {
		return isvalid.InvalidError.Errorf("Ballot has empty Fact hash")
	}
	if blt.FactSignature() == nil {
		return isvalid.InvalidError.Errorf("Ballot has empty FactSignature()")
	}

	if err := blt.Signer().Verify(util.ConcatBytesSlice(blt.Fact().Hash().Bytes(), b), blt.FactSignature()); err != nil {
		return err
	}

	return blt.Signer().Verify(
		util.ConcatBytesSlice(blt.Fact().Hash().Bytes(), b),
		blt.FactSignature(),
	)
}

func SignBaseBallotV0(blt Ballot, bb BaseBallotV0, pk key.Privatekey, networkID []byte) (BaseBallotV0, error) {
	if err := bb.IsReadyToSign(networkID); err != nil {
		return BaseBallotV0{}, err
	}

	var bodyHash valuehash.Hash
	if h, err := blt.GenerateBodyHash(); err != nil {
		return BaseBallotV0{}, err
	} else {
		bodyHash = h
	}

	var sig key.Signature
	if s, err := pk.Sign(util.ConcatBytesSlice(bodyHash.Bytes(), networkID)); err != nil {
		return BaseBallotV0{}, err
	} else {
		sig = s
	}

	factSig, err := pk.Sign(util.ConcatBytesSlice(blt.Fact().Hash().Bytes(), networkID))
	if err != nil {
		return BaseBallotV0{}, err
	}

	bb.signer = pk.Publickey()
	bb.signature = sig
	bb.signedAt = localtime.Now()
	bb.bodyHash = bodyHash
	bb.factSignature = factSig

	return bb, nil
}

func GenerateHash(blt Ballot, bb BaseBallotV0, bs ...[]byte) valuehash.Hash {
	bl := util.ConcatBytesSlice(bb.Bytes(), blt.Fact().Bytes())
	if len(bs) > 0 {
		bl = util.ConcatBytesSlice(
			bl,
			util.ConcatBytesSlice(bs...),
		)
	}

	return valuehash.NewSHA256(bl)
}
