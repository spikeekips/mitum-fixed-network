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
	INITV0Hint     = hint.NewHint(INITType, "v0.0.1")
	INITFactV0Hint = hint.NewHint(INITFactType, "v0.0.1")
)

type INITFactV0 struct {
	BaseFactV0
	previousBlock valuehash.Hash
}

func NewINITFactV0(
	height base.Height,
	round base.Round,
	previousBlock valuehash.Hash,
) INITFactV0 {
	return INITFactV0{
		BaseFactV0:    NewBaseFactV0(height, round),
		previousBlock: previousBlock,
	}
}

func (INITFactV0) Hint() hint.Hint {
	return INITFactV0Hint
}

func (ibf INITFactV0) IsValid(networkID []byte) error {
	return isvalid.Check([]isvalid.IsValider{
		ibf.BaseFactV0,
		ibf.previousBlock,
	}, networkID, false)
}

func (ibf INITFactV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(ibf.Bytes())
}

func (ibf INITFactV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		ibf.BaseFactV0.Bytes(),
		ibf.previousBlock.Bytes(),
	)
}

func (ibf INITFactV0) PreviousBlock() valuehash.Hash {
	return ibf.previousBlock
}

type INITV0 struct {
	BaseBallotV0
	INITFactV0
	voteproof       base.Voteproof
	acceptVoteproof base.Voteproof
}

func NewINITV0(
	node base.Address,
	height base.Height,
	round base.Round,
	previousBlock valuehash.Hash,
	voteproof base.Voteproof,
	acceptVoteproof base.Voteproof,
) INITV0 {
	return INITV0{
		BaseBallotV0: NewBaseBallotV0(node),
		INITFactV0: NewINITFactV0(
			height,
			round,
			previousBlock,
		),
		voteproof:       voteproof,
		acceptVoteproof: acceptVoteproof,
	}
}

func (ib INITV0) Hash() valuehash.Hash {
	return ib.BaseBallotV0.Hash()
}

func (INITV0) Hint() hint.Hint {
	return INITV0Hint
}

func (INITV0) Stage() base.Stage {
	return base.StageINIT
}

func (ib INITV0) IsValid(networkID []byte) error {
	if ib.Height() == base.Height(0) {
		if ib.voteproof != nil {
			return xerrors.Errorf("not empty voteproof for genesis INITBallot")
		}

		if err := isvalid.Check([]isvalid.IsValider{
			ib.BaseBallotV0,
			ib.INITFactV0,
		}, networkID, false); err != nil {
			return err
		}
	}

	if ib.voteproof == nil || ib.acceptVoteproof == nil {
		return xerrors.Errorf("empty voteproof")
	}

	if err := isvalid.Check([]isvalid.IsValider{
		ib.BaseBallotV0,
		ib.INITFactV0,
		ib.voteproof,
		ib.acceptVoteproof,
	}, networkID, false); err != nil {
		return err
	}

	return IsValidBallot(ib, networkID)
}

func (ib INITV0) Voteproof() base.Voteproof {
	return ib.voteproof
}

func (ib INITV0) ACCEPTVoteproof() base.Voteproof {
	return ib.acceptVoteproof
}

func (ib INITV0) GenerateHash() valuehash.Hash {
	return GenerateHash(ib, ib.BaseBallotV0)
}

func (ib INITV0) GenerateBodyHash() (valuehash.Hash, error) {
	if err := ib.INITFactV0.IsValid(nil); err != nil {
		return nil, err
	}

	bs := make([][]byte, 3)
	bs[0] = ib.INITFactV0.Bytes()

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

func (ib INITV0) Fact() base.Fact {
	return ib.INITFactV0
}

func (ib *INITV0) Sign(pk key.Privatekey, networkID []byte) error {
	newBase, err := SignBaseBallotV0(ib, ib.BaseBallotV0, pk, networkID)
	if err != nil {
		return err
	}

	ib.BaseBallotV0 = newBase
	ib.BaseBallotV0 = ib.BaseBallotV0.SetHash(ib.GenerateHash())

	return nil
}
