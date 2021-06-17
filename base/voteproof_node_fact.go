package base

import (
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	BaseVoteproofNodeFactType = hint.Type("base-voteproof-node-fact")
	BaseVoteproofNodeFactHint = hint.NewHint(BaseVoteproofNodeFactType, "v0.0.1")
)

type BaseVoteproofNodeFact struct {
	address       Address
	ballot        valuehash.Hash
	fact          valuehash.Hash
	factSignature key.Signature
	signer        key.Publickey
}

func NewBaseVoteproofNodeFact(
	address Address,
	blt valuehash.Hash,
	fact valuehash.Hash,
	factSignature key.Signature,
	signer key.Publickey,
) VoteproofNodeFact {
	return BaseVoteproofNodeFact{
		address:       address,
		ballot:        blt,
		fact:          fact,
		factSignature: factSignature,
		signer:        signer,
	}
}

func (BaseVoteproofNodeFact) Hint() hint.Hint {
	return BaseVoteproofNodeFactHint
}

func (vf BaseVoteproofNodeFact) IsValid(networkID []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vf.address,
		vf.ballot,
		vf.fact,
		vf.factSignature,
		vf.signer,
	}, nil, false); err != nil {
		return err
	}

	return vf.signer.Verify(util.ConcatBytesSlice(vf.fact.Bytes(), networkID), vf.factSignature)
}

func (vf BaseVoteproofNodeFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		vf.address.Bytes(),
		vf.ballot.Bytes(),
		vf.fact.Bytes(),
		vf.factSignature.Bytes(),
		vf.signer.Bytes(),
	)
}

func (vf BaseVoteproofNodeFact) Ballot() valuehash.Hash {
	return vf.ballot
}

func (vf BaseVoteproofNodeFact) Fact() valuehash.Hash {
	return vf.fact
}

func (vf BaseVoteproofNodeFact) Signature() key.Signature {
	return vf.factSignature
}

func (vf BaseVoteproofNodeFact) Node() Address {
	return vf.address
}

func (vf BaseVoteproofNodeFact) Signer() key.Publickey {
	return vf.signer
}
