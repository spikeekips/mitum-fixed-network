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
	SIGNV0Hint     = hint.NewHint(SIGNType, "v0.0.1")
	SIGNFactV0Hint = hint.NewHint(SIGNFactType, "v0.0.1")
)

type SIGNFactV0 struct {
	BaseFactV0
	proposal valuehash.Hash
	newBlock valuehash.Hash
}

func (SIGNFactV0) Hint() hint.Hint {
	return SIGNFactV0Hint
}

func (sbf SIGNFactV0) IsValid(networkID []byte) error {
	return isvalid.Check([]isvalid.IsValider{
		sbf.BaseFactV0,
		sbf.proposal,
		sbf.newBlock,
	}, networkID, false)
}

func (sbf SIGNFactV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(sbf.Bytes())
}

func (sbf SIGNFactV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		sbf.BaseFactV0.Bytes(),
		sbf.proposal.Bytes(),
		sbf.newBlock.Bytes(),
	)
}

func (sbf SIGNFactV0) Proposal() valuehash.Hash {
	return sbf.proposal
}

func (sbf SIGNFactV0) NewBlock() valuehash.Hash {
	return sbf.newBlock
}

type SIGNV0 struct {
	BaseBallotV0
	SIGNFactV0
}

func NewSIGNV0(
	node base.Address,
	height base.Height,
	round base.Round,
	proposal valuehash.Hash,
	newBlock valuehash.Hash,
) SIGNV0 {
	return SIGNV0{
		BaseBallotV0: BaseBallotV0{
			node: node,
		},
		SIGNFactV0: SIGNFactV0{
			BaseFactV0: BaseFactV0{
				height: height,
				round:  round,
			},
			proposal: proposal,
			newBlock: newBlock,
		},
	}
}

func (sb SIGNV0) Hash() valuehash.Hash {
	return sb.BaseBallotV0.Hash()
}

func (SIGNV0) Hint() hint.Hint {
	return SIGNV0Hint
}

func (SIGNV0) Stage() base.Stage {
	return base.StageSIGN
}

func (sb SIGNV0) IsValid(networkID []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		sb.BaseBallotV0,
		sb.SIGNFactV0,
	}, networkID, false); err != nil {
		return err
	}

	return IsValidBallot(sb, networkID)
}

func (sb SIGNV0) GenerateHash() valuehash.Hash {
	return GenerateHash(sb, sb.BaseBallotV0)
}

func (sb SIGNV0) GenerateBodyHash() (valuehash.Hash, error) {
	if err := sb.SIGNFactV0.IsValid(nil); err != nil {
		return nil, err
	}

	return valuehash.NewSHA256(sb.SIGNFactV0.Bytes()), nil
}

func (sb SIGNV0) Fact() base.Fact {
	return sb.SIGNFactV0
}

func (sb *SIGNV0) Sign(pk key.Privatekey, networkID []byte) error {
	newBase, err := SignBaseBallotV0(sb, sb.BaseBallotV0, pk, networkID)
	if err != nil {
		return err
	}

	sb.BaseBallotV0 = newBase
	sb.BaseBallotV0 = sb.BaseBallotV0.SetHash(sb.GenerateHash())

	return nil
}
