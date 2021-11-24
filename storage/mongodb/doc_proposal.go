package mongodbstorage

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

type ProposalDoc struct {
	BaseDoc
	proposal base.Proposal
}

func NewProposalDoc(proposal base.Proposal, enc encoder.Encoder) (ProposalDoc, error) {
	b, err := NewBaseDoc(proposal.Fact().Hash().String(), proposal, enc)
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

	fact := pd.proposal.Fact()
	m["hash_string"] = fact.Hash().String()
	m["height"] = fact.Height()
	m["round"] = fact.Round()
	m["proposer"] = fact.Proposer().String()

	return bsonenc.Marshal(m)
}

func loadProposalFromDecoder(decoder func(interface{}) error, encs *encoder.Encoders) (base.Proposal, error) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return nil, err
	}

	_, hinter, err := LoadDataFromDoc(b, encs)
	if err != nil {
		return nil, err
	}

	i, ok := hinter.(base.Proposal)
	if !ok {
		return nil, errors.Errorf("not Proposal: %T", hinter)
	}

	return i, nil
}
