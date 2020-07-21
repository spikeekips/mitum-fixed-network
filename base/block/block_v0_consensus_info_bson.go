package block

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (bc BlockConsensusInfoV0) MarshalBSON() ([]byte, error) {
	m := bson.M{}
	if bc.initVoteproof != nil {
		m["init_voteproof"] = bc.initVoteproof
	}

	if bc.acceptVoteproof != nil {
		m["accept_voteproof"] = bc.acceptVoteproof
	}

	if bc.suffrageInfo != nil {
		m["suffrage_info"] = bc.suffrageInfo
	}

	return bsonenc.Marshal(bsonenc.MergeBSONM(bsonenc.NewHintedDoc(bc.Hint()), m))
}

type BlockConsensusInfoV0UnpackBSON struct {
	IV bson.Raw `bson:"init_voteproof,omitempty"`
	AV bson.Raw `bson:"accept_voteproof,omitempty"`
	SI bson.Raw `bson:"suffrage_info,omitempty"`
}

func (bc *BlockConsensusInfoV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var nbc BlockConsensusInfoV0UnpackBSON
	if err := enc.Unmarshal(b, &nbc); err != nil {
		return err
	}

	return bc.unpack(enc, nbc.IV, nbc.AV, nbc.SI)
}

func (si SuffrageInfoV0) MarshalBSON() ([]byte, error) {
	m := bson.M{
		"proposer": si.proposer,
		"nodes":    si.nodes,
	}

	return bsonenc.Marshal(bsonenc.MergeBSONM(bsonenc.NewHintedDoc(si.Hint()), m))
}

type SuffrageInfoV0UnpackBSON struct {
	PR base.AddressDecoder `bson:"proposer"`
	NS []bson.Raw          `bson:"nodes"`
}

func (si *SuffrageInfoV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var nsi SuffrageInfoV0UnpackBSON
	if err := enc.Unmarshal(b, &nsi); err != nil {
		return err
	}

	bsn := make([][]byte, len(nsi.NS))
	for i := range nsi.NS {
		bsn[i] = nsi.NS[i]
	}

	return si.unpack(enc, nsi.PR, bsn)
}
