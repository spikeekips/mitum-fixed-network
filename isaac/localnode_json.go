package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
)

func (ln *LocalNode) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		AD  base.Address   `json:"address"`
		PUK key.Publickey  `json:"publickey"`
		PRK key.Privatekey `json:"privatekey"`
	}{
		AD:  ln.address,
		PUK: ln.publickey,
		PRK: ln.privatekey,
	})
}
