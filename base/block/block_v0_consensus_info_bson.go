package block

import (
	"github.com/spikeekips/mitum/util/encoder"
	"go.mongodb.org/mongo-driver/bson"
)

func (bc BlockConsensusInfoV0) MarshalBSON() ([]byte, error) {
	m := bson.M{}
	if bc.initVoteproof != nil {
		m["init_voteproof"] = bc.initVoteproof
	}

	if bc.acceptVoteproof != nil {
		m["accept_voteproof"] = bc.acceptVoteproof
	}

	return bson.Marshal(encoder.MergeBSONM(encoder.NewBSONHintedDoc(bc.Hint()), m))
}

type BlockConsensusInfoV0UnpackBSON struct {
	IV bson.Raw `bson:"init_voteproof,omitempty"`
	AV bson.Raw `bson:"accept_voteproof,omitempty"`
}

func (bc *BlockConsensusInfoV0) UnpackBSON(b []byte, enc *encoder.BSONEncoder) error {
	var nbc BlockConsensusInfoV0UnpackBSON
	if err := enc.Unmarshal(b, &nbc); err != nil {
		return err
	}

	return bc.unpack(enc, nbc.IV, nbc.AV)
}
