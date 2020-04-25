package mongodbstorage

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/encoder"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
)

type VoteproofDoc struct {
	BaseDoc
	voteproof base.Voteproof
}

func NewVoteproofDoc(voteproof base.Voteproof, enc encoder.Encoder) (VoteproofDoc, error) {
	b, err := NewBaseDoc(nil, voteproof, enc)
	if err != nil {
		return VoteproofDoc{}, err
	}

	return VoteproofDoc{
		BaseDoc:   b,
		voteproof: voteproof,
	}, nil
}

func (vd VoteproofDoc) MarshalBSON() ([]byte, error) {
	m, err := vd.BaseDoc.M()
	if err != nil {
		return nil, err
	}

	m["stage"] = vd.voteproof.Stage()
	m["height"] = vd.voteproof.Height()
	m["round"] = vd.voteproof.Round()

	return bson.Marshal(m)
}

func loadVoteproofFromDecoder(decoder func(interface{}) error, encs *encoder.Encoders) (base.Voteproof, error) {
	var b bson.Raw
	if err := decoder(&b); err != nil {
		return nil, err
	}

	var voteproof base.Voteproof

	_, hinter, err := loadWithEncoder(b, encs)
	if err != nil {
		return nil, err
	} else if i, ok := hinter.(base.Voteproof); !ok {
		return nil, xerrors.Errorf("not Block: %T", hinter)
	} else {
		voteproof = i
	}

	return voteproof, nil
}
