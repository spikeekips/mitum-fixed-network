package ballot

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	INITBallotV0Hint     = hint.NewHint(INITBallotType, "v0.0.1")
	INITBallotFactV0Hint = hint.NewHint(INITBallotFactType, "v0.0.1")
)

type INITBallotFactV0 struct {
	BaseBallotFactV0
	previousBlock valuehash.Hash
}

func NewINITBallotFactV0(
	height base.Height,
	round base.Round,
	previousBlock valuehash.Hash,
) INITBallotFactV0 {
	return INITBallotFactV0{
		BaseBallotFactV0: NewBaseBallotFactV0(height, round),
		previousBlock:    previousBlock,
	}
}

func (INITBallotFactV0) Hint() hint.Hint {
	return INITBallotFactV0Hint
}

func (ibf INITBallotFactV0) IsValid(networkID []byte) error {
	return isvalid.Check([]isvalid.IsValider{
		ibf.BaseBallotFactV0,
		ibf.previousBlock,
	}, networkID, false)
}

func (ibf INITBallotFactV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(ibf.Bytes())
}

func (ibf INITBallotFactV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		ibf.BaseBallotFactV0.Bytes(),
		ibf.previousBlock.Bytes(),
	)
}

func (ibf INITBallotFactV0) PreviousBlock() valuehash.Hash {
	return ibf.previousBlock
}

type INITBallotV0 struct {
	BaseBallotV0
	INITBallotFactV0
	voteproof       base.Voteproof
	acceptVoteproof base.Voteproof
}

func NewINITBallotV0(
	node base.Address,
	height base.Height,
	round base.Round,
	previousBlock valuehash.Hash,
	voteproof base.Voteproof,
	acceptVoteproof base.Voteproof,
) INITBallotV0 {
	return INITBallotV0{
		BaseBallotV0: NewBaseBallotV0(node),
		INITBallotFactV0: NewINITBallotFactV0(
			height,
			round,
			previousBlock,
		),
		voteproof:       voteproof,
		acceptVoteproof: acceptVoteproof,
	}
}

func (ib INITBallotV0) Hash() valuehash.Hash {
	return ib.BaseBallotV0.Hash()
}

func (INITBallotV0) Hint() hint.Hint {
	return INITBallotV0Hint
}

func (INITBallotV0) Stage() base.Stage {
	return base.StageINIT
}

func (ib INITBallotV0) IsValid(networkID []byte) error {
	if ib.Height() == base.Height(0) {
		if ib.voteproof != nil {
			return xerrors.Errorf("not empty voteproof for genesis INITBallot")
		}

		if err := isvalid.Check([]isvalid.IsValider{
			ib.BaseBallotV0,
			ib.INITBallotFactV0,
		}, networkID, false); err != nil {
			return err
		}
	}

	if ib.voteproof == nil || ib.acceptVoteproof == nil {
		return xerrors.Errorf("empty voteproof")
	}

	if err := isvalid.Check([]isvalid.IsValider{
		ib.BaseBallotV0,
		ib.INITBallotFactV0,
		ib.voteproof,
		ib.acceptVoteproof,
	}, networkID, false); err != nil {
		return err
	}

	return IsValidBallot(ib, networkID)
}

func (ib INITBallotV0) Voteproof() base.Voteproof {
	return ib.voteproof
}

func (ib INITBallotV0) ACCEPTVoteproof() base.Voteproof {
	return ib.acceptVoteproof
}

func (ib INITBallotV0) GenerateHash() valuehash.Hash {
	return GenerateHash(ib, ib.BaseBallotV0)
}

func (ib INITBallotV0) GenerateBodyHash() (valuehash.Hash, error) {
	if err := ib.INITBallotFactV0.IsValid(nil); err != nil {
		return nil, err
	}

	bs := make([][]byte, 3)
	bs[0] = ib.INITBallotFactV0.Bytes()

	if ib.Height() != base.Height(0) {
		if ib.voteproof != nil {
			bs[1] = ib.voteproof.Bytes()
		}
		if ib.acceptVoteproof != nil {
			bs[2] = ib.acceptVoteproof.Bytes()
		}
	}

	return valuehash.NewSHA256(util.ConcatBytesSlice(bs...)), nil
}

func (ib INITBallotV0) Fact() base.Fact {
	return ib.INITBallotFactV0
}

func (ib *INITBallotV0) Sign(pk key.Privatekey, networkID []byte) error {
	newBase, err := SignBaseBallotV0(ib, ib.BaseBallotV0, pk, networkID)
	if err != nil {
		return err
	}

	ib.BaseBallotV0 = newBase
	ib.BaseBallotV0 = ib.BaseBallotV0.SetHash(ib.GenerateHash())

	return nil
}
