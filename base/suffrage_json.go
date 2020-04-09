package base

import (
	"github.com/spikeekips/mitum/util"
)

func (as ActingSuffrage) MarshalJSON() ([]byte, error) {
	nodes := make([]string, len(as.nodes))
	var index int
	for n := range as.nodes {
		nodes[index] = n.String()
		index++
	}

	return util.JSONMarshal(struct {
		H Height   `json:"height"`
		R Round    `json:"round"`
		P string   `json:"proposer"`
		N []string `json:"nodes"`
	}{
		H: as.height,
		R: as.round,
		P: as.proposer.Address().String(),
		N: nodes,
	})
}
