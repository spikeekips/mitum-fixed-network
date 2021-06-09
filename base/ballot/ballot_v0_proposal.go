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
	ProposalV0Hint     = hint.NewHint(ProposalBallotType, "v0.0.1")
	ProposalFactV0Hint = hint.NewHint(ProposalBallotFactType, "v0.0.1")
)

type ProposalFactV0 struct {
	BaseBallotFactV0
	seals []valuehash.Hash
}

func (prf ProposalFactV0) Hint() hint.Hint {
	return ProposalFactV0Hint
}

func (prf ProposalFactV0) IsValid(networkID []byte) error {
	if err := prf.BaseBallotFactV0.IsValid(networkID); err != nil {
		return err
	}

	if err := isvalid.Check(func() []isvalid.IsValider {
		var vs []isvalid.IsValider
		for _, s := range prf.seals {
			vs = append(vs, s)
		}

		return vs
	}(), networkID, false); err != nil {
		return err
	}

	// NOTE duplicated seals will not be allowed
	{
		founds := map[string]struct{}{}
		for _, h := range prf.seals {
			if _, found := founds[h.String()]; found {
				return xerrors.Errorf("duplicated Seal found in Proposal")
			}

			founds[h.String()] = struct{}{}
		}
	}

	return nil
}

func (prf ProposalFactV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(prf.Bytes())
}

func (prf ProposalFactV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		prf.BaseBallotFactV0.Bytes(),
		func() []byte {
			hl := make([][]byte, len(prf.seals))
			for i := range prf.seals {
				hl[i] = prf.seals[i].Bytes()
			}

			return util.ConcatBytesSlice(hl...)
		}(),
	)
}

func (prf ProposalFactV0) Seals() []valuehash.Hash {
	return prf.seals
}

type ProposalV0 struct {
	BaseBallotV0
	ProposalFactV0
	voteproof base.Voteproof
}

func NewProposalV0(
	node base.Address,
	height base.Height,
	round base.Round,
	seals []valuehash.Hash,
	voteproof base.Voteproof,
) ProposalV0 {
	return ProposalV0{
		BaseBallotV0: BaseBallotV0{
			node: node,
		},
		ProposalFactV0: ProposalFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: height,
				round:  round,
			},
			seals: seals,
		},
		voteproof: voteproof,
	}
}

func (pr ProposalV0) Hash() valuehash.Hash {
	return pr.BaseBallotV0.Hash()
}

func (pr ProposalV0) Hint() hint.Hint {
	return ProposalV0Hint
}

func (pr ProposalV0) Stage() base.Stage {
	return base.StageProposal
}

func (pr ProposalV0) IsValid(networkID []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		pr.BaseBallotV0,
		pr.ProposalFactV0,
	}, networkID, false); err != nil {
		return err
	}

	return IsValidBallot(pr, networkID)
}

func (pr ProposalV0) Voteproof() base.Voteproof {
	return pr.voteproof
}

func (pr ProposalV0) GenerateHash() valuehash.Hash {
	return GenerateHash(pr, pr.BaseBallotV0)
}

func (pr ProposalV0) GenerateBodyHash() (valuehash.Hash, error) {
	if err := pr.ProposalFactV0.IsValid(nil); err != nil {
		return nil, err
	}

	bs := make([][]byte, 2)
	bs[0] = pr.ProposalFactV0.Bytes()

	if pr.Height() != base.Height(0) {
		if pr.voteproof != nil {
			bs[1] = pr.voteproof.Bytes()
		}
	}

	return valuehash.NewSHA256(util.ConcatBytesSlice(bs...)), nil
}

func (pr ProposalV0) Fact() base.Fact {
	return pr.ProposalFactV0
}

func (pr *ProposalV0) Sign(pk key.Privatekey, networkID []byte) error {
	if newBase, err := SignBaseBallotV0(pr, pr.BaseBallotV0, pk, networkID); err != nil {
		return err
	} else {
		pr.BaseBallotV0 = newBase
		pr.BaseBallotV0 = pr.BaseBallotV0.SetHash(pr.GenerateHash())
	}

	return nil
}
