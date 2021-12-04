package ballot

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

type BaseFact struct {
	hint.BaseHinter
	h      valuehash.Hash
	height base.Height
	round  base.Round
}

func NewBaseFact(ht hint.Hint, height base.Height, round base.Round) BaseFact {
	return BaseFact{
		BaseHinter: hint.NewBaseHinter(ht),
		height:     height,
		round:      round,
	}
}

func (fact BaseFact) bytes() []byte {
	return util.ConcatBytesSlice(
		fact.height.Bytes(),
		fact.round.Bytes(),
	)
}

func (fact BaseFact) IsValid([]byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		fact.BaseHinter,
		fact.h,
		fact.height,
	}, nil, false); err != nil {
		return fmt.Errorf("invalid ballot fact: %w", err)
	}

	if fact.height <= base.PreGenesisHeight {
		return isvalid.InvalidError.Errorf("invalid height, %q of ballot fact", fact.height)
	}

	return nil
}

func (fact BaseFact) Hash() valuehash.Hash {
	return fact.h
}

func (fact BaseFact) Stage() base.Stage {
	switch fact.BaseHinter.Hint().Type() {
	case base.INITBallotFactType:
		return base.StageINIT
	case base.ProposalFactType:
		return base.StageProposal
	case base.ACCEPTBallotFactType:
		return base.StageACCEPT
	default:
		return base.Stage(0)
	}
}

func (fact BaseFact) Height() base.Height {
	return fact.height
}

func (fact BaseFact) Round() base.Round {
	return fact.round
}

func isValidFact(fact base.BallotFact) error {
	if fact == nil {
		return errors.Errorf("nil fact")
	}

	var bf BaseFact
	var bb []byte
	switch t := fact.(type) {
	case INITFact:
		bf = t.BaseFact
		bb = t.bytes()
	case ProposalFact:
		bf = t.BaseFact
		bb = t.bytes()
	case ACCEPTFact:
		bf = t.BaseFact
		bb = t.bytes()
	default:
		return isvalid.InvalidError.Errorf("unknown ballot fact, %T", fact)
	}

	if err := bf.IsValid(nil); err != nil {
		return fmt.Errorf("invalid ballot fact, %T: %w", fact, err)
	}

	if h := valuehash.NewSHA256(bb); !fact.Hash().Equal(h) {
		return isvalid.InvalidError.Errorf("ballot fact hash does not match")
	}

	return nil
}
