package ballot

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	SIGNBallotV0Hint     hint.Hint = hint.MustHint(SIGNBallotType, "0.0.1")
	SIGNBallotFactV0Hint hint.Hint = hint.MustHint(SIGNBallotFactType, "0.0.1")
)

type SIGNBallotFactV0 struct {
	BaseBallotFactV0
	proposal valuehash.Hash
	newBlock valuehash.Hash
}

func (sbf SIGNBallotFactV0) Hint() hint.Hint {
	return SIGNBallotFactV0Hint
}

func (sbf SIGNBallotFactV0) IsValid(networkID []byte) error {
	return isvalid.Check([]isvalid.IsValider{
		sbf.BaseBallotFactV0,
		sbf.proposal,
		sbf.newBlock,
	}, networkID, false)
}

func (sbf SIGNBallotFactV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(sbf.Bytes())
}

func (sbf SIGNBallotFactV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		sbf.BaseBallotFactV0.Bytes(),
		sbf.proposal.Bytes(),
		sbf.newBlock.Bytes(),
	)
}

func (sbf SIGNBallotFactV0) Proposal() valuehash.Hash {
	return sbf.proposal
}

func (sbf SIGNBallotFactV0) NewBlock() valuehash.Hash {
	return sbf.newBlock
}

type SIGNBallotV0 struct {
	BaseBallotV0
	SIGNBallotFactV0
}

func NewSIGNBallotV0(
	node base.Address,
	height base.Height,
	round base.Round,
	proposal valuehash.Hash,
	newBlock valuehash.Hash,
) SIGNBallotV0 {
	return SIGNBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: node,
		},
		SIGNBallotFactV0: SIGNBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: height,
				round:  round,
			},
			proposal: proposal,
			newBlock: newBlock,
		},
	}
}

func (sb SIGNBallotV0) Hash() valuehash.Hash {
	return sb.BaseBallotV0.Hash()
}

func (sb SIGNBallotV0) Hint() hint.Hint {
	return SIGNBallotV0Hint
}

func (sb SIGNBallotV0) Stage() base.Stage {
	return base.StageSIGN
}

func (sb SIGNBallotV0) IsValid(networkID []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		sb.BaseBallotV0,
		sb.SIGNBallotFactV0,
	}, networkID, false); err != nil {
		return err
	}

	return IsValidBallot(sb, networkID)
}

func (sb SIGNBallotV0) GenerateHash() valuehash.Hash {
	return GenerateHash(sb, sb.BaseBallotV0)
}

func (sb SIGNBallotV0) GenerateBodyHash() (valuehash.Hash, error) {
	if err := sb.SIGNBallotFactV0.IsValid(nil); err != nil {
		return nil, err
	}

	return valuehash.NewSHA256(sb.SIGNBallotFactV0.Bytes()), nil
}

func (sb SIGNBallotV0) Fact() base.Fact {
	return sb.SIGNBallotFactV0
}

func (sb *SIGNBallotV0) Sign(pk key.Privatekey, networkID []byte) error {
	if newBase, err := SignBaseBallotV0(sb, sb.BaseBallotV0, pk, networkID); err != nil {
		return err
	} else {
		sb.BaseBallotV0 = newBase
		sb.BaseBallotV0 = sb.BaseBallotV0.SetHash(sb.GenerateHash())
	}

	return nil
}
