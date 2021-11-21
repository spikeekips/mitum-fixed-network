package ballot

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	INITType         = hint.Type("init-ballot")
	ProposalType     = hint.Type("proposal")
	SIGNType         = hint.Type("sign-ballot")
	ACCEPTType       = hint.Type("accept-ballot")
	INITFactType     = hint.Type("init-ballot-fact")
	ProposalFactType = hint.Type("proposal-fact")
	SIGNFactType     = hint.Type("sign-ballot-fact")
	ACCEPTFactType   = hint.Type("accept-ballot-fact")
)

type Ballot interface {
	zerolog.LogObjectMarshaler
	seal.Seal
	Fact() base.Fact
	FactSignature() key.Signature
	Stage() base.Stage
	Height() base.Height
	Round() base.Round
	Node() base.Address
}

type INIT interface {
	Ballot
	PreviousBlock() valuehash.Hash
	Voteproof() base.Voteproof
	ACCEPTVoteproof() base.Voteproof
}

type Proposal interface {
	Ballot
	Voteproof() base.Voteproof
	Seals() []valuehash.Hash // NOTE collection of received Seals, which must have Operations()
}

type SIGN interface {
	Ballot
	Proposal() valuehash.Hash
	NewBlock() valuehash.Hash
}

type ACCEPT interface {
	Ballot
	Proposal() valuehash.Hash
	NewBlock() valuehash.Hash
	Voteproof() base.Voteproof
}

type INITFact interface {
	valuehash.Hasher
	PreviousBlock() valuehash.Hash
}

type SIGNFact interface {
	valuehash.Hasher
	Proposal() valuehash.Hash
	NewBlock() valuehash.Hash
}

type ACCEPTFact interface {
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

	if i, ok := blt.(base.Voteproofer); ok {
		if err := IsValidVoteproof(blt, i.Voteproof()); err != nil {
			return err
		}
	}

	if err := blt.Signer().Verify(util.ConcatBytesSlice(blt.Fact().Hash().Bytes(), b), blt.FactSignature()); err != nil {
		return err
	}

	return blt.Signer().Verify(
		util.ConcatBytesSlice(blt.Fact().Hash().Bytes(), b),
		blt.FactSignature(),
	)
}

func IsValidVoteproof(blt Ballot, voteproof base.Voteproof) error {
	if !voteproof.IsFinished() {
		return errors.Errorf("not yet finished voteproof found in ballot")
	}

	switch t := blt.(type) {
	case INIT:
		if err := isValidVoteproofInINIT(t, voteproof); err != nil {
			return err
		}

		return isValidACCEPTVoteproofInINIT(t, t.ACCEPTVoteproof())
	case ACCEPT:
		return isValidVoteproofInACCEPT(t, voteproof)
	case Proposal:
		return isValidVoteproofInProposal(t, voteproof)
	default:
		return errors.Errorf("not supported voteproof in ballot, %T", blt)
	}
}

func isValidVoteproofInINIT(blt INIT, voteproof base.Voteproof) error {
	vs := voteproof.Stage()
	if vs != base.StageINIT && vs != base.StageACCEPT {
		return errors.Errorf("invalid voteproof stage for init ballot; it should be init or accept, not %v", vs)
	}

	bh := blt.Height()
	vh := voteproof.Height()
	br := blt.Round()
	vr := voteproof.Round()

	switch vs {
	case base.StageINIT:
		if bh != vh {
			return errors.Errorf("different height of init ballot + init voteproof; ballot=%v voteproof=%v", bh, vh)
		} else if br != vr+1 {
			return errors.Errorf("wrong round of init ballot + init voteproof; ballot=%v voteproof=%v+1", br, vr)
		}
	case base.StageACCEPT:
		if voteproof.Result() == base.VoteResultDraw {
			switch {
			case bh != vh:
				return errors.Errorf(
					"different height of init ballot + draw accept voteproof; ballot=%v voteproof=%v", bh, vh)
			case br == base.Round(0):
				return errors.Errorf("wrong round of init ballot + draw accept voteproof; round 0")
			case br != vr+1:
				return errors.Errorf("wrong round of init ballot + init voteproof; ballot=%v voteproof=%v+1", br, vr)
			}
		} else if bh != vh+1 {
			return errors.Errorf("wrong height of init ballot + accept voteproof; ballot=%v voteproof=%v+1",
				bh, vh)
		}
	}

	return nil
}

func isValidACCEPTVoteproofInINIT(blt INIT, voteproof base.Voteproof) error {
	if vs := voteproof.Stage(); vs != base.StageACCEPT {
		return errors.Errorf("invalid accept voteproof stage for init ballot; it should be accept, not %v", vs)
	}

	bh := blt.Height()
	vh := voteproof.Height()

	if bh != vh+1 {
		return errors.Errorf("wrong height of init ballot + accept voteproof; ballot=%v voteproof=%v+1",
			bh, vh)
	}

	return nil
}

func isValidVoteproofInACCEPT(blt ACCEPT, voteproof base.Voteproof) error {
	if vs := voteproof.Stage(); vs != base.StageINIT {
		return errors.Errorf("invalid voteproof stage for accept ballot; it should be init, not %v", vs)
	} else if voteproof.Result() != base.VoteResultMajority {
		return errors.Errorf(
			"invalid init voteproof result for accept ballot; it should be majority, not %v", voteproof.Result())
	}

	if blt.Height() != voteproof.Height() {
		return errors.Errorf("accept ballot has different height with init voteproof; ballot=%v voteproof=%v",
			blt.Height(), voteproof.Height())
	} else if blt.Round() != voteproof.Round() {
		return errors.Errorf("accept ballot has different round with init voteproof; ballot=%v voteproof=%v",
			blt.Round(), voteproof.Round())
	}

	return nil
}

func isValidVoteproofInProposal(blt Proposal, voteproof base.Voteproof) error {
	if vs := voteproof.Stage(); vs != base.StageINIT {
		return errors.Errorf("invalid voteproof stage for proposal; it should be init, not %v", vs)
	} else if voteproof.Result() != base.VoteResultMajority {
		return errors.Errorf(
			"invalid init voteproof result for proposal; it should be majority, not %v", voteproof.Result())
	}

	if blt.Height() != voteproof.Height() {
		return errors.Errorf("proposal has different height with init voteproof; ballot=%v voteproof=%v",
			blt.Height(), voteproof.Height())
	} else if blt.Round() != voteproof.Round() {
		return errors.Errorf("proposal has different round with init voteproof; ballot=%v voteproof=%v",
			blt.Round(), voteproof.Round())
	}

	return nil
}

func SignBaseBallotV0(blt Ballot, bb BaseBallotV0, pk key.Privatekey, networkID []byte) (BaseBallotV0, error) {
	if err := bb.IsReadyToSign(networkID); err != nil {
		return BaseBallotV0{}, err
	}

	bodyHash, err := blt.GenerateBodyHash()
	if err != nil {
		return BaseBallotV0{}, err
	}

	sig, err := pk.Sign(util.ConcatBytesSlice(bodyHash.Bytes(), networkID))
	if err != nil {
		return BaseBallotV0{}, err
	}

	factSig, err := pk.Sign(util.ConcatBytesSlice(blt.Fact().Hash().Bytes(), networkID))
	if err != nil {
		return BaseBallotV0{}, err
	}

	bb.signer = pk.Publickey()
	bb.signature = sig
	bb.signedAt = localtime.UTCNow()
	bb.bodyHash = bodyHash
	bb.factSignature = factSig

	return bb, nil
}

func GenerateHash(blt Ballot, bb BaseBallotV0, bs ...[]byte) valuehash.Hash {
	bl := util.ConcatBytesSlice(bb.Bytes(), blt.Fact().Hash().Bytes())
	if len(bs) > 0 {
		bl = util.ConcatBytesSlice(
			bl,
			util.ConcatBytesSlice(bs...),
		)
	}

	return valuehash.NewSHA256(bl)
}
