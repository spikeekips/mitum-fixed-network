package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

var (
	ACCEPTBallotV0Hint     hint.Hint = hint.MustHint(ACCEPTBallotType, "0.0.1")
	ACCEPTBallotFactV0Hint hint.Hint = hint.MustHint(ACCEPTBallotFactType, "0.0.1")
)

type ACCEPTBallotFactV0 struct {
	BaseBallotFactV0
	proposal valuehash.Hash
	newBlock valuehash.Hash
}

func (abf ACCEPTBallotFactV0) Hint() hint.Hint {
	return ACCEPTBallotFactV0Hint
}

func (abf ACCEPTBallotFactV0) IsValid([]byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		abf.BaseBallotFactV0,
		abf.proposal,
		abf.newBlock,
	}, nil, false); err != nil {
		return err
	}

	return nil
}

func (abf ACCEPTBallotFactV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(abf.Bytes())
}

func (abf ACCEPTBallotFactV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		abf.BaseBallotFactV0.Bytes(),
		abf.proposal.Bytes(),
		abf.newBlock.Bytes(),
	)
}

func (abf ACCEPTBallotFactV0) Proposal() valuehash.Hash {
	return abf.proposal
}

func (abf ACCEPTBallotFactV0) NewBlock() valuehash.Hash {
	return abf.newBlock
}

// TODO ACCEPTBallot should have SIGNBallots

type ACCEPTBallotV0 struct {
	BaseBallotV0
	ACCEPTBallotFactV0
	voteproof Voteproof
}

func NewACCEPTBallotV0(
	localstate *Localstate,
	height Height,
	round Round,
	newBlock Block,
	initVoteproof Voteproof,
	networkID []byte,
) (ACCEPTBallotV0, error) {
	ab := ACCEPTBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: localstate.Node().Address(),
		},
		ACCEPTBallotFactV0: ACCEPTBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: height,
				round:  round,
			},
			proposal: newBlock.Proposal(),
			newBlock: newBlock.Hash(),
		},
		voteproof: initVoteproof,
	}

	if err := ab.Sign(localstate.Node().Privatekey(), networkID); err != nil {
		return ACCEPTBallotV0{}, err
	}

	return ab, nil
}

func NewACCEPTBallotV0FromLocalstate(
	localstate *Localstate,
	round Round,
	newBlock Block,
) (ACCEPTBallotV0, error) {
	lastBlock := localstate.LastBlock()
	if lastBlock == nil {
		return ACCEPTBallotV0{}, xerrors.Errorf("lastBlock is empty")
	}

	ab := ACCEPTBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: localstate.Node().Address(),
		},
		ACCEPTBallotFactV0: ACCEPTBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: lastBlock.Height() + 1,
				round:  round,
			},
			proposal: newBlock.Proposal(),
			newBlock: newBlock.Hash(),
		},
		voteproof: localstate.LastINITVoteproof(),
	}

	if err := ab.Sign(localstate.Node().Privatekey(), localstate.Policy().NetworkID()); err != nil {
		return ACCEPTBallotV0{}, err
	}

	return ab, nil
}

func (ab ACCEPTBallotV0) Hash() valuehash.Hash {
	return ab.BaseBallotV0.Hash()
}

func (ab ACCEPTBallotV0) Hint() hint.Hint {
	return ACCEPTBallotV0Hint
}

func (ab ACCEPTBallotV0) Stage() Stage {
	return StageACCEPT
}

func (ab ACCEPTBallotV0) IsValid(b []byte) error {
	if ab.voteproof == nil {
		return xerrors.Errorf("empty Voteproof")
	}

	if err := isvalid.Check([]isvalid.IsValider{
		ab.BaseBallotV0,
		ab.ACCEPTBallotFactV0,
		ab.voteproof,
	}, b, false); err != nil {
		return err
	}

	if err := IsValidBallot(ab, b); err != nil {
		return err
	}

	return nil
}

func (ab ACCEPTBallotV0) Voteproof() Voteproof {
	return ab.voteproof
}

func (ab ACCEPTBallotV0) GenerateHash() (valuehash.Hash, error) {
	e := util.ConcatBytesSlice(
		ab.BaseBallotV0.Bytes(),
		ab.ACCEPTBallotFactV0.Bytes(),
		ab.voteproof.Bytes(),
	)

	return valuehash.NewSHA256(e), nil
}

func (ab ACCEPTBallotV0) GenerateBodyHash() (valuehash.Hash, error) {
	if err := ab.ACCEPTBallotFactV0.IsValid(nil); err != nil {
		return nil, err
	}

	e := util.ConcatBytesSlice(
		ab.ACCEPTBallotFactV0.Bytes(),
		ab.voteproof.Bytes(),
	)

	return valuehash.NewSHA256(e), nil
}

func (ab ACCEPTBallotV0) Fact() operation.Fact {
	return ab.ACCEPTBallotFactV0
}

func (ab *ACCEPTBallotV0) Sign(pk key.Privatekey, b []byte) error { // nolint
	if err := ab.BaseBallotV0.IsReadyToSign(b); err != nil {
		return err
	}

	var bodyHash valuehash.Hash
	if h, err := ab.GenerateBodyHash(); err != nil {
		return err
	} else {
		bodyHash = h
	}

	var sig key.Signature
	if s, err := pk.Sign(util.ConcatBytesSlice(bodyHash.Bytes(), b)); err != nil {
		return err
	} else {
		sig = s
	}

	factHash := ab.ACCEPTBallotFactV0.Hash()
	factSig, err := pk.Sign(util.ConcatBytesSlice(factHash.Bytes(), b))
	if err != nil {
		return err
	}

	ab.BaseBallotV0.signer = pk.Publickey()
	ab.BaseBallotV0.signature = sig
	ab.BaseBallotV0.signedAt = localtime.Now()
	ab.BaseBallotV0.bodyHash = bodyHash
	ab.BaseBallotV0.factHash = factHash
	ab.BaseBallotV0.factSignature = factSig

	if h, err := ab.GenerateHash(); err != nil {
		return err
	} else {
		ab.BaseBallotV0 = ab.BaseBallotV0.SetHash(h)
	}

	return nil
}
