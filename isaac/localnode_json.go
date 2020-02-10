package isaac

import (
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/util"
)

func (ln LocalNode) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		AD  Address        `json:"address"`
		PUK key.Publickey  `json:"publickey"`
		PRK key.Privatekey `json:"privatekey"`
	}{
		AD:  ln.address,
		PUK: ln.publickey,
		PRK: ln.privatekey,
	})
}
