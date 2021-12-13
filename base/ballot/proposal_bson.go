package ballot

import (
	"time"

	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (fact ProposalFact) MarshalBSON() ([]byte, error) {
	return bson.Marshal(bsonenc.MergeBSONM(
		fact.BaseFact.packerBSON(),
		bson.M{
			"proposer":    fact.proposer,
			"operations":  fact.ops,
			"proposed_at": fact.proposedAt,
		}))
}

type ProposalFactUnpackerBSON struct {
	PR  base.AddressDecoder `bson:"proposer"`
	OPS []valuehash.Bytes   `bson:"operations"`
	PA  time.Time           `bson:"proposed_at"`
}

func (fact *ProposalFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	if err := fact.BaseFact.unpackBSON(b, enc); err != nil {
		return err
	}

	var uf ProposalFactUnpackerBSON
	if err := enc.Unmarshal(b, &uf); err != nil {
		return err
	}

	pr, err := uf.PR.Encode(enc)
	if err != nil {
		return err
	}
	fact.proposer = pr

	ops := make([]valuehash.Hash, len(uf.OPS))
	for i := range uf.OPS {
		ops[i] = uf.OPS[i]
	}

	fact.ops = ops
	fact.proposedAt = uf.PA

	return nil
}
