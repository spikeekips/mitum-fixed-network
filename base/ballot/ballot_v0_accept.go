package ballot

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	ACCEPTV0Hint     = hint.NewHint(ACCEPTType, "v0.0.1")
	ACCEPTFactV0Hint = hint.NewHint(ACCEPTFactType, "v0.0.1")
)

type ACCEPTFactV0 struct {
	BaseFactV0
	proposal valuehash.Hash
	newBlock valuehash.Hash
}

func (ACCEPTFactV0) Hint() hint.Hint {
	return ACCEPTFactV0Hint
}

func (abf ACCEPTFactV0) IsValid(networkID []byte) error {
	return isvalid.Check([]isvalid.IsValider{
		abf.BaseFactV0,
		abf.proposal,
		abf.newBlock,
	}, networkID, false)
}

func (abf ACCEPTFactV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(abf.Bytes())
}

func (abf ACCEPTFactV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		abf.BaseFactV0.Bytes(),
		abf.proposal.Bytes(),
		abf.newBlock.Bytes(),
	)
}

func (abf ACCEPTFactV0) Proposal() valuehash.Hash {
	return abf.proposal
}

func (abf ACCEPTFactV0) NewBlock() valuehash.Hash {
	return abf.newBlock
}

// FUTURE ACCEPT Ballot should have SIGN Ballots

type ACCEPTV0 struct {
	BaseBallotV0
	ACCEPTFactV0
	voteproof base.Voteproof
}

func NewACCEPTV0(
	node base.Address,
	height base.Height,
	round base.Round,
	proposal valuehash.Hash,
	newBlock valuehash.Hash,
	voteproof base.Voteproof,
) ACCEPTV0 {
	return ACCEPTV0{
		BaseBallotV0: BaseBallotV0{
			node: node,
		},
		ACCEPTFactV0: ACCEPTFactV0{
			BaseFactV0: BaseFactV0{
				height: height,
				round:  round,
			},
			proposal: proposal,
			newBlock: newBlock,
		},
		voteproof: voteproof,
	}
}

func (ab ACCEPTV0) Hash() valuehash.Hash {
	return ab.BaseBallotV0.Hash()
}

func (ACCEPTV0) Hint() hint.Hint {
	return ACCEPTV0Hint
}

func (ACCEPTV0) Stage() base.Stage {
	return base.StageACCEPT
}

func (ab ACCEPTV0) IsValid(networkID []byte) error {
	if ab.voteproof == nil {
		return errors.Errorf("empty Voteproof")
	}

	if err := isvalid.Check([]isvalid.IsValider{
		ab.BaseBallotV0,
		ab.ACCEPTFactV0,
		ab.voteproof,
	}, networkID, false); err != nil {
		return err
	}

	return IsValidBallot(ab, networkID)
}

func (ab ACCEPTV0) Voteproof() base.Voteproof {
	return ab.voteproof
}

func (ab ACCEPTV0) GenerateHash() valuehash.Hash {
	var vb []byte
	if ab.voteproof != nil {
		vb = ab.voteproof.Bytes()
	}

	return GenerateHash(ab, ab.BaseBallotV0, vb)
}

func (ab ACCEPTV0) GenerateBodyHash() (valuehash.Hash, error) {
	if err := ab.ACCEPTFactV0.IsValid(nil); err != nil {
		return nil, err
	}

	var vb []byte
	if ab.voteproof != nil {
		vb = ab.voteproof.Bytes()
	}

	return valuehash.NewSHA256(util.ConcatBytesSlice(ab.ACCEPTFactV0.Bytes(), vb)), nil
}

func (ab ACCEPTV0) Fact() base.Fact {
	return ab.ACCEPTFactV0
}

func (ab *ACCEPTV0) Sign(pk key.Privatekey, networkID []byte) error {
	newBase, err := SignBaseBallotV0(ab, ab.BaseBallotV0, pk, networkID)
	if err != nil {
		return err
	}

	ab.BaseBallotV0 = newBase
	ab.BaseBallotV0 = ab.BaseBallotV0.SetHash(ab.GenerateHash())

	return nil
}
