package base

import jsonencoder "github.com/spikeekips/mitum/util/encoder/json"

type ActingSuffragePacker struct {
	H Height   `json:"height" bson:"height"`
	R Round    `json:"round" bson:"round"`
	P string   `json:"proposer" bson:"proposer"`
	N []string `json:"nodes" bson:"nodes"`
}

func (as ActingSuffrage) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(ActingSuffragePacker{
		H: as.height,
		R: as.round,
		P: as.proposer.Address().String(),
		N: as.NodesSlice(),
	})
}
