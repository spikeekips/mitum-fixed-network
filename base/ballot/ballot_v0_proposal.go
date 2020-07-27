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
	ProposalV0Hint     hint.Hint = hint.MustHint(ProposalBallotType, "0.0.1")
	ProposalFactV0Hint hint.Hint = hint.MustHint(ProposalBallotFactType, "0.0.1")
)

type ProposalFactV0 struct {
	BaseBallotFactV0
	facts []valuehash.Hash
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
		for _, s := range prf.facts {
			vs = append(vs, s)
		}
		for _, s := range prf.seals {
			vs = append(vs, s)
		}

		return vs
	}(), networkID, false); err != nil {
		return err
	}

	// NOTE duplicated facts or seals will not be allowed
	{
		mo := map[string]struct{}{}
		for _, h := range prf.facts {
			if _, found := mo[h.String()]; found {
				return xerrors.Errorf("duplicated Operation found in Proposal")
			}

			mo[h.String()] = struct{}{}
		}
	}

	{
		mo := map[string]struct{}{}
		for _, h := range prf.seals {
			if _, found := mo[h.String()]; found {
				return xerrors.Errorf("duplicated Seal found in Proposal")
			}

			mo[h.String()] = struct{}{}
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
			hl := make([][]byte, len(prf.facts)+len(prf.seals))
			for i := range prf.facts {
				hl[i] = prf.facts[i].Bytes()
			}
			for i := range prf.seals {
				hl[len(prf.facts)+i] = prf.seals[i].Bytes()
			}

			return util.ConcatBytesSlice(hl...)
		}(),
	)
}

func (prf ProposalFactV0) Facts() []valuehash.Hash {
	return prf.facts
}

func (prf ProposalFactV0) Seals() []valuehash.Hash {
	return prf.seals
}

type ProposalV0 struct {
	BaseBallotV0
	ProposalFactV0
}

func NewProposalV0(
	node base.Address,
	height base.Height,
	round base.Round,
	facts []valuehash.Hash,
	seals []valuehash.Hash,
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
			facts: facts,
			seals: seals,
		},
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

func (pr ProposalV0) GenerateHash() valuehash.Hash {
	return GenerateHash(pr, pr.BaseBallotV0)
}

func (pr ProposalV0) GenerateBodyHash() (valuehash.Hash, error) {
	if err := pr.ProposalFactV0.IsValid(nil); err != nil {
		return nil, err
	}

	return valuehash.NewSHA256(pr.ProposalFactV0.Bytes()), nil
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
