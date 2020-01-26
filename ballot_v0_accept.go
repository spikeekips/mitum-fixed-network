package mitum

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

var ACCEPTBallotV0Hint hint.Hint = hint.MustHint(ACCEPTBallotType, "0.1")

type ACCEPTBallotV0 struct {
	BaseBallotV0
	h  valuehash.Hash
	bh valuehash.Hash

	//-x-------------------- hashing parts
	proposal   valuehash.Hash
	newBlock   valuehash.Hash
	voteResult VoteResult
	//--------------------x-
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
	return ab.bh
}

func (ab ACCEPTBallotV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{ab.BaseBallotV0, ab.proposal, ab.newBlock}, b); err != nil {
		return err
	}

	return nil
}

func (ab ACCEPTBallotV0) Proposal() valuehash.Hash {
	return ab.proposal
}

func (ab ACCEPTBallotV0) NewBlock() valuehash.Hash {
	return ab.newBlock
}

func (ab ACCEPTBallotV0) VoteResult() VoteResult {
	return ab.voteResult
}

func (ab ACCEPTBallotV0) GenerateHash(b []byte) (valuehash.Hash, error) {
	if err := ab.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		ab.BaseBallotV0.Bytes(),
		ab.proposal.Bytes(),
		ab.newBlock.Bytes(),
		ab.voteResult.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (ab ACCEPTBallotV0) GenerateBodyHash(b []byte) (valuehash.Hash, error) {
	if err := ab.BaseBallotV0.IsReadyToSign(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		ab.BaseBallotV0.BodyBytes(),
		ab.proposal.Bytes(),
		ab.newBlock.Bytes(),
		ab.voteResult.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (ab *ACCEPTBallotV0) Sign(pk key.Privatekey, b []byte) error {
	var bodyHash valuehash.Hash
	if h, err := ab.GenerateBodyHash(b); err != nil {
		return err
	} else {
		bodyHash = h
	}

	sig, err := pk.Sign(util.ConcatSlice([][]byte{bodyHash.Bytes(), b}))
	if err != nil {
		return err
	}
	ab.BaseBallotV0.signer = pk.Publickey()
	ab.BaseBallotV0.signature = sig
	ab.BaseBallotV0.signedAt = localtime.Now()
	ab.bh = bodyHash

	if h, err := ab.GenerateHash(b); err != nil {
		return err
	} else {
		ab.h = h
	}

	return nil
}
