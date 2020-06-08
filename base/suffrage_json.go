package base

import jsonenc "github.com/spikeekips/mitum/util/encoder/json"

type ActingSuffragePacker struct {
	H Height    `json:"height" bson:"height"`
	R Round     `json:"round" bson:"round"`
	P string    `json:"proposer" bson:"proposer"`
	N []Address `json:"nodes" bson:"nodes"`
}

func (as ActingSuffrage) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(ActingSuffragePacker{
		H: as.height,
		R: as.round,
		P: as.proposer.String(),
		N: as.Nodes(),
	})
}
