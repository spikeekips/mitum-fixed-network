package mitum

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

var ProposalV0Hint hint.Hint = hint.MustHint(ProposalBallotType, "0.1")

type ProposalV0Fact struct {
	BaseBallotFactV0
	seals []valuehash.Hash
}

func (prf ProposalV0Fact) IsValid(b []byte) error {
	if err := prf.BaseBallotFactV0.IsValid(b); err != nil {
		return err
	}

	return nil
}

func (prf ProposalV0Fact) Hash(b []byte) (valuehash.Hash, error) {
	// TODO check IsValid?
	e := util.ConcatSlice([][]byte{prf.Bytes(), b})

	return valuehash.NewSHA256(e), nil
}

func (prf ProposalV0Fact) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		prf.BaseBallotFactV0.Bytes(),
		func() []byte {
			var hl [][]byte
			for _, h := range prf.seals {
				hl = append(hl, h.Bytes())
			}

			return util.ConcatSlice(hl)
		}(),
	})
}

func (prf ProposalV0Fact) Seals() []valuehash.Hash {
	return prf.seals
}

type ProposalV0 struct {
	BaseBallotV0
	ProposalV0Fact
	h             valuehash.Hash
	bodyHash      valuehash.Hash
	factHash      valuehash.Hash
	factSignature key.Signature
}

func (pr ProposalV0) Hint() hint.Hint {
	return ProposalV0Hint
}

func (pr ProposalV0) Stage() Stage {
	return StageProposal
}

func (pr ProposalV0) Hash() valuehash.Hash {
	return pr.h
}

func (pr ProposalV0) BodyHash() valuehash.Hash {
	return pr.bodyHash
}

func (pr ProposalV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		pr.BaseBallotV0,
		pr.ProposalV0Fact,
	}, b); err != nil {
		return err
	}

	return nil
}

func (pr ProposalV0) GenerateHash(b []byte) (valuehash.Hash, error) {
	if err := pr.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		pr.BaseBallotV0.Bytes(),
		pr.ProposalV0Fact.Bytes(),
		pr.bodyHash.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (pr ProposalV0) GenerateBodyHash(b []byte) (valuehash.Hash, error) {
	if err := pr.ProposalV0Fact.IsValid(b); err != nil {
		return nil, err
	}

	e := util.ConcatSlice([][]byte{
		pr.ProposalV0Fact.Bytes(),
		b,
	})

	return valuehash.NewSHA256(e), nil
}

func (pr ProposalV0) Fact() Fact {
	return pr.ProposalV0Fact
}

func (pr ProposalV0) FactHash() valuehash.Hash {
	return pr.factHash
}

func (pr ProposalV0) FactSignature() key.Signature {
	return pr.factSignature
}

func (pr *ProposalV0) Sign(pk key.Privatekey, b []byte) error { // nolint
	if err := pr.BaseBallotV0.IsReadyToSign(b); err != nil {
		return err
	}

	var bodyHash valuehash.Hash
	if h, err := pr.GenerateBodyHash(b); err != nil {
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
	if h, err := pr.ProposalV0Fact.Hash(b); err != nil {
		return err
	} else {
		factHash = h
	}

	factSig, err := pk.Sign(util.ConcatSlice([][]byte{factHash.Bytes(), b}))
	if err != nil {
		return err
	}

	pr.BaseBallotV0.signer = pk.Publickey()
	pr.BaseBallotV0.signature = sig
	pr.BaseBallotV0.signedAt = localtime.Now()
	pr.bodyHash = bodyHash
	pr.factHash = factHash
	pr.factSignature = factSig

	if h, err := pr.GenerateHash(b); err != nil {
		return err
	} else {
		pr.h = h
	}

	return nil
}
