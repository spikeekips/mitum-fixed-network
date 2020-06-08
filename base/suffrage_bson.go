package base

import bsonenc "github.com/spikeekips/mitum/util/encoder/bson"

func (as ActingSuffrage) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(ActingSuffragePacker{
		H: as.height,
		R: as.round,
		P: as.proposer.String(),
		N: as.Nodes(),
	})
}
