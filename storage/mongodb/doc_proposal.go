package mongodbstorage

import (
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/util/encoder"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
)

type ProposalDoc struct {
	BaseDoc
	proposal ballot.Proposal
}

func NewProposalDoc(proposal ballot.Proposal, enc encoder.Encoder) (ProposalDoc, error) {
	b, err := NewBaseDoc(proposal.Hash(), proposal, enc)
	if err != nil {
		return ProposalDoc{}, err
	}

	return ProposalDoc{
		BaseDoc:  b,
		proposal: proposal,
	}, nil
}

func (pd ProposalDoc) MarshalBSON() ([]byte, error) {
	m, err := pd.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["height"] = pd.proposal.Height()
	m["round"] = pd.proposal.Round()

	return bson.Marshal(m)
}

func loadProposalFromDecoder(decoder func(interface{}) error, encs *encoder.Encoders) (ballot.Proposal, error) {
	sl, err := loadSealFromDecoder(decoder, encs)
	if err != nil {
		return nil, err
	}

	var proposal ballot.Proposal
	if i, ok := sl.(ballot.Proposal); !ok {
		return nil, xerrors.Errorf("not Proposal: %T", sl)
	} else {
		proposal = i
	}

	return proposal, nil
}
