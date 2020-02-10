package isaac

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
	"golang.org/x/xerrors"
)

var ACCEPTBallotV0Hint hint.Hint = hint.MustHint(ACCEPTBallotType, "0.1")

type ACCEPTBallotFactV0 struct {
	BaseBallotFactV0
	proposal valuehash.Hash
	newBlock valuehash.Hash
}

func (abf ACCEPTBallotFactV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		abf.BaseBallotFactV0,
		abf.proposal,
		abf.newBlock,
	}, b); err != nil {
		return err
	}

	return nil
}

func (abf ACCEPTBallotFactV0) Hash(b []byte) (valuehash.Hash, error) {
	// TODO check IsValid?
	e := util.ConcatSlice([][]byte{abf.Bytes(), b})

	// TODO create common hash func for hashing Ballot.
	return valuehash.NewSHA256(e), nil
}

func (abf ACCEPTBallotFactV0) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		abf.BaseBallotFactV0.Bytes(),
		abf.proposal.Bytes(),
		abf.newBlock.Bytes(),
	})
}

func (abf ACCEPTBallotFactV0) Proposal() valuehash.Hash {
	return abf.proposal
}

func (abf ACCEPTBallotFactV0) NewBlock() valuehash.Hash {
	return abf.newBlock
}

type ACCEPTBallotV0 struct {
	BaseBallotV0
	ACCEPTBallotFactV0
	h             valuehash.Hash
	bodyHash      valuehash.Hash
	voteProof     VoteProof
	factHash      valuehash.Hash
	factSignature key.Signature
}

func NewACCEPTBallotV0FromLocalState(
	localState *LocalState,
	round Round,
	newBlock Block,
	b []byte,
) (ACCEPTBallotV0, error) {
	lastBlock := localState.LastBlock()
	if lastBlock == nil {
		return ACCEPTBallotV0{}, xerrors.Errorf("lastBlock is empty")
	}

	ab := ACCEPTBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: localState.Node().Address(),
		},
		ACCEPTBallotFactV0: ACCEPTBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: lastBlock.Height() + 1,
				round:  round,
			},
			proposal: newBlock.Proposal(),
			newBlock: newBlock.Hash(),
		},
		voteProof: localState.LastINITVoteProof(),
	}

	// TODO NetworkID must be given.
	if err := ab.Sign(localState.Node().Privatekey(), b); err != nil {
		return ACCEPTBallotV0{}, err
	}

	return ab, nil
}

func (ab ACCEPTBallotV0) Hint() hint.Hint {
	return ACCEPTBallotV0Hint
}

func (ab ACCEPTBallotV0) Stage() Stage {
	return StageACCEPT
}

func (ab ACCEPTBallotV0) Hash() valuehash.Hash {
	return ab.h
}

func (ab ACCEPTBallotV0) BodyHash() valuehash.Hash {
	return ab.bodyHash
}

func (ab ACCEPTBallotV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		ab.BaseBallotV0,
		ab.ACCEPTBallotFactV0,
	}, b); err != nil {
		return err
	}

	return nil
}

func (ab ACCEPTBallotV0) VoteProof() VoteProof {
	return ab.voteProof
}

func (ab ACCEPTBallotV0) GenerateHash(b []byte) (valuehash.Hash, error) {
	if err := ab.IsValid(b); err != nil {
		return nil, err
	}

	var vpb []byte
	if ab.voteProof != nil {
		vpb = ab.voteProof.Bytes()
	}

	e := util.ConcatSlice([][]byte{
		ab.BaseBallotV0.Bytes(),
		ab.ACCEPTBallotFactV0.Bytes(),
		ab.bodyHash.Bytes(),
		vpb,
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (ab ACCEPTBallotV0) GenerateBodyHash(b []byte) (valuehash.Hash, error) {
	if err := ab.ACCEPTBallotFactV0.IsValid(b); err != nil {
		return nil, err
	}

	var vpb []byte
	if ab.voteProof != nil {
		vpb = ab.voteProof.Bytes()
	}

	e := util.ConcatSlice([][]byte{
		ab.ACCEPTBallotFactV0.Bytes(),
		vpb,
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (ab ACCEPTBallotV0) Fact() Fact {
	return ab.ACCEPTBallotFactV0
}

func (ab ACCEPTBallotV0) FactHash() valuehash.Hash {
	return ab.factHash
}

func (ab ACCEPTBallotV0) FactSignature() key.Signature {
	return ab.factSignature
}

func (ab *ACCEPTBallotV0) Sign(pk key.Privatekey, b []byte) error { // nolint
	if err := ab.BaseBallotV0.IsReadyToSign(b); err != nil {
		return err
	}

	var bodyHash valuehash.Hash
	if h, err := ab.GenerateBodyHash(b); err != nil {
		return err
	} else {
		bodyHash = h
	}

	var sig key.Signature
	if s, err := pk.Sign(util.ConcatSlice([][]byte{bodyHash.Bytes(), b})); err != nil {
		return err
	} else {
		sig = s
	}

	var factHash valuehash.Hash
	if h, err := ab.ACCEPTBallotFactV0.Hash(b); err != nil {
		return err
	} else {
		factHash = h
	}

	factSig, err := pk.Sign(util.ConcatSlice([][]byte{factHash.Bytes(), b}))
	if err != nil {
		return err
	}

	ab.BaseBallotV0.signer = pk.Publickey()
	ab.BaseBallotV0.signature = sig
	ab.BaseBallotV0.signedAt = localtime.Now()
	ab.bodyHash = bodyHash
	ab.factHash = factHash
	ab.factSignature = factSig

	if h, err := ab.GenerateHash(b); err != nil {
		return err
	} else {
		ab.h = h
	}

	return nil
}
