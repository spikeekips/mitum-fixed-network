package ballot

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	ProposalFactHint   = hint.NewHint(base.ProposalFactType, "v0.0.1")
	ProposalFactHinter = ProposalFact{BaseFact: BaseFact{BaseHinter: hint.NewBaseHinter(ProposalFactHint)}}
	ProposalHint       = hint.NewHint(base.ProposalType, "v0.0.1")
	ProposalHinter     = Proposal{BaseSeal: BaseSeal{BaseSeal: seal.NewBaseSealWithHint(ProposalHint)}}
)

type ProposalFact struct {
	BaseFact
	proposer   base.Address
	ops        []valuehash.Hash
	proposedAt time.Time
}

func NewProposalFact(
	height base.Height,
	round base.Round,
	proposer base.Address,
	ops []valuehash.Hash,
) ProposalFact {
	fact := ProposalFact{
		BaseFact: NewBaseFact(
			ProposalFactHint,
			height,
			round,
		),
		proposer:   proposer,
		ops:        ops,
		proposedAt: localtime.UTCNow(),
	}

	fact.BaseFact.h = valuehash.NewSHA256(fact.bytes())

	return fact
}

func (fact ProposalFact) Proposer() base.Address {
	return fact.proposer
}

func (fact ProposalFact) Operations() []valuehash.Hash {
	return fact.ops
}

func (fact ProposalFact) ProposedAt() time.Time {
	return fact.proposedAt
}

func (fact ProposalFact) IsValid([]byte) error {
	if fact.proposedAt.IsZero() {
		return isvalid.InvalidError.Errorf("empty proposed at")
	}

	if err := isValidFact(fact); err != nil {
		return err
	}

	if err := isvalid.Check(nil, false, fact.proposer); err != nil {
		return err
	}

	founds := map[string]struct{}{}
	for i := range fact.ops {
		h := fact.ops[i]
		if h == nil {
			return isvalid.InvalidError.Errorf("empty operation hash found in proposal")
		}

		if err := h.IsValid(nil); err != nil {
			return isvalid.InvalidError.Errorf("invalid operation hash found in proposal: %w", err)
		}

		if _, found := founds[h.String()]; found {
			return isvalid.InvalidError.Errorf("duplicated operaiotn found in proposal")
		}

		founds[h.String()] = struct{}{}
	}

	return nil
}

func (fact ProposalFact) bytes() []byte {
	var b []byte
	if fact.proposer != nil {
		b = fact.proposer.Bytes()
	}

	return util.ConcatBytesSlice(
		fact.BaseFact.bytes(),
		b,
		func() []byte {
			hs := make([][]byte, len(fact.ops))
			for i := range fact.ops {
				if fact.ops[i] == nil {
					continue
				}

				hs[i] = fact.ops[i].Bytes()
			}

			return util.ConcatBytesSlice(hs...)
		}(),
		localtime.NewTime(fact.proposedAt).Bytes(),
	)
}

type Proposal struct {
	BaseSeal
}

func NewProposal(
	fact ProposalFact,
	n base.Address,
	baseVoteproof base.Voteproof,
	pk key.Privatekey,
	networkID base.NetworkID,
) (Proposal, error) {
	b, err := NewBaseSeal(ProposalHint, fact, n, baseVoteproof, nil, pk, networkID)
	if err != nil {
		return Proposal{}, err
	}

	return Proposal{BaseSeal: b}, nil
}

func (sl Proposal) Fact() base.ProposalFact {
	return sl.RawFact().(base.ProposalFact)
}

func (sl Proposal) IsValid(networkID []byte) error {
	if err := sl.BaseSeal.IsValid(networkID); err != nil {
		return isvalid.InvalidError.Errorf("invalid proposal: %w", err)
	}

	if _, ok := sl.Fact().(ProposalFact); !ok {
		return isvalid.InvalidError.Errorf("invalid fact of proposal; %T", sl.Fact())
	}

	if sl.FactSign().SignedAt().Before(sl.Fact().(ProposalFact).proposedAt) {
		return isvalid.InvalidError.Errorf("proposal is signed at before proposed at; %q < %q",
			sl.FactSign().SignedAt(), sl.Fact().(ProposalFact).proposedAt)
	}

	if err := sl.isValidBaseVoteproofAfterINIT(); err != nil {
		return isvalid.InvalidError.Errorf("invalid proposal: %w", err)
	}

	return nil
}
