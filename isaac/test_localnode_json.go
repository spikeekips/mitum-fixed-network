// +build test

package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

func (ln *LocalNode) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(struct {
		jsonencoder.HintedHead
		AD  base.Address   `json:"address"`
		PUK key.Publickey  `json:"publickey"`
		PRK key.Privatekey `json:"privatekey"`
	}{
		HintedHead: jsonencoder.NewHintedHead(ln.Hint()),
		AD:         ln.Address(),
		PUK:        ln.Publickey(),
		PRK:        ln.privatekey,
	})
}
