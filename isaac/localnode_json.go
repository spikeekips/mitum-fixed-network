package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

func (ln *LocalNode) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(struct {
		AD  base.Address   `json:"address"`
		PUK key.Publickey  `json:"publickey"`
		PRK key.Privatekey `json:"privatekey"`
	}{
		AD:  ln.address,
		PUK: ln.publickey,
		PRK: ln.privatekey,
	})
}
