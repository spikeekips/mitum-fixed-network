package ballot

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
)

var (
	ProposalV0Hint     hint.Hint = hint.MustHint(ProposalBallotType, "0.0.1")
	ProposalFactV0Hint hint.Hint = hint.MustHint(ProposalBallotFactType, "0.0.1")
)

type ProposalFactV0 struct {
	BaseBallotFactV0
	operations []valuehash.Hash
	seals      []valuehash.Hash
}

func (prf ProposalFactV0) Hint() hint.Hint {
	return ProposalFactV0Hint
}

func (prf ProposalFactV0) IsValid(b []byte) error {
	if err := prf.BaseBallotFactV0.IsValid(b); err != nil {
		return err
	}

	if err := isvalid.Check(func() []isvalid.IsValider {
		var vs []isvalid.IsValider
		for _, s := range prf.operations {
			vs = append(vs, s)
		}
		for _, s := range prf.seals {
			vs = append(vs, s)
		}

		return vs
	}(), b, false); err != nil {
		return err
	}

	// NOTE duplicated operations or seals will not be allowed
	{
		mo := map[valuehash.Hash]struct{}{}
		for _, h := range prf.operations {
			if _, found := mo[h]; found {
				return xerrors.Errorf("duplicated Operation found in Proposal")
			}

			mo[h] = struct{}{}
		}
	}

	{
		mo := map[valuehash.Hash]struct{}{}
		for _, h := range prf.seals {
			if _, found := mo[h]; found {
				return xerrors.Errorf("duplicated Seal found in Proposal")
			}

			mo[h] = struct{}{}
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
			var hl [][]byte
			for _, h := range prf.operations {
				hl = append(hl, h.Bytes())
			}
			for _, h := range prf.seals {
				hl = append(hl, h.Bytes())
			}

			return util.ConcatBytesSlice(hl...)
		}(),
	)
}

func (prf ProposalFactV0) Operations() []valuehash.Hash {
	return prf.operations
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
	operations []valuehash.Hash,
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
			operations: operations,
			seals:      seals,
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

func (pr ProposalV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		pr.BaseBallotV0,
		pr.ProposalFactV0,
	}, b, false); err != nil {
		return err
	}

	if err := IsValidBallot(pr, b); err != nil {
		return err
	}

	return nil
}

func (pr ProposalV0) GenerateHash() (valuehash.Hash, error) {
	return valuehash.NewSHA256(util.ConcatBytesSlice(pr.BaseBallotV0.Bytes(), pr.ProposalFactV0.Bytes())), nil
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

func (pr *ProposalV0) Sign(pk key.Privatekey, b []byte) error { // nolint
	if err := pr.BaseBallotV0.IsReadyToSign(b); err != nil {
		return err
	}

	var bodyHash valuehash.Hash
	if h, err := pr.GenerateBodyHash(); err != nil {
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

	factHash := pr.ProposalFactV0.Hash()
	factSig, err := pk.Sign(util.ConcatBytesSlice(factHash.Bytes(), b))
	if err != nil {
		return err
	}

	pr.BaseBallotV0.signer = pk.Publickey()
	pr.BaseBallotV0.signature = sig
	pr.BaseBallotV0.signedAt = localtime.Now()
	pr.BaseBallotV0.bodyHash = bodyHash
	pr.BaseBallotV0.factHash = factHash
	pr.BaseBallotV0.factSignature = factSig

	if h, err := pr.GenerateHash(); err != nil {
		return err
	} else {
		pr.BaseBallotV0 = pr.BaseBallotV0.SetHash(h)
	}

	return nil
}
