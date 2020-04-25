package base

import "go.mongodb.org/mongo-driver/bson"

func (as ActingSuffrage) MarshalBSON() ([]byte, error) {
	return bson.Marshal(ActingSuffragePacker{
		H: as.height,
		R: as.round,
		P: as.proposer.Address().String(),
		N: as.NodesSlice(),
	})
}
