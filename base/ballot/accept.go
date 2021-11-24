package ballot // nolint:dupl

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	ACCEPTFactHint   = hint.NewHint(base.ACCEPTBallotFactType, "v0.0.1")
	ACCEPTFactHinter = ACCEPTFact{BaseFact: BaseFact{hint: ACCEPTFactHint}}
	ACCEPTHint       = hint.NewHint(base.ACCEPTBallotType, "v0.0.1")
	ACCEPTHinter     = ACCEPT{BaseSeal: BaseSeal{BaseSeal: seal.NewBaseSealWithHint(ACCEPTHint)}}
)

type ACCEPTFact struct {
	BaseFact
	proposal valuehash.Hash
	newBlock valuehash.Hash
}

func NewACCEPTFact(
	height base.Height,
	round base.Round,
	proposal,
	newBlock valuehash.Hash,
) ACCEPTFact {
	fact := ACCEPTFact{
		BaseFact: NewBaseFact(
			ACCEPTFactHint,
			height,
			round,
		),
		proposal: proposal,
		newBlock: newBlock,
	}

	fact.BaseFact.h = valuehash.NewSHA256(fact.bytes())

	return fact
}

func (fact ACCEPTFact) Proposal() valuehash.Hash {
	return fact.proposal
}

func (fact ACCEPTFact) NewBlock() valuehash.Hash {
	return fact.newBlock
}

func (fact ACCEPTFact) IsValid([]byte) error {
	if err := isValidFact(fact); err != nil {
		return err
	}

	return isvalid.Check([]isvalid.IsValider{
		fact.proposal,
		fact.newBlock,
	}, nil, false)
}

func (fact ACCEPTFact) bytes() []byte {
	var bp, bb []byte
	if fact.proposal != nil {
		bp = fact.proposal.Bytes()
	}

	if fact.newBlock != nil {
		bp = fact.newBlock.Bytes()
	}

	return util.ConcatBytesSlice(fact.BaseFact.bytes(), bp, bb)
}

type ACCEPT struct {
	BaseSeal
}

func NewACCEPT(
	fact ACCEPTFact,
	n base.Address,
	baseVoteproof base.Voteproof,
	pk key.Privatekey,
	networkID base.NetworkID,
) (ACCEPT, error) {
	b, err := NewBaseSeal(ACCEPTHint, fact, n, baseVoteproof, nil, pk, networkID)
	if err != nil {
		return ACCEPT{}, err
	}

	return ACCEPT{BaseSeal: b}, nil
}

func (sl ACCEPT) Fact() base.ACCEPTBallotFact {
	return sl.RawFact().(base.ACCEPTBallotFact)
}

func (sl ACCEPT) IsValid(networkID []byte) error {
	if err := sl.BaseSeal.IsValid(networkID); err != nil {
		return fmt.Errorf("invalid proposal: %w", err)
	}

	if _, ok := sl.Fact().(ACCEPTFact); !ok {
		return errors.Errorf("invalid fact of accept ballot; %T", sl.Fact())
	}

	if err := sl.isValidBaseVoteproofAfterINIT(); err != nil {
		return isvalid.InvalidError.Errorf("invalid proposal: %w", err)
	}

	return nil
}
