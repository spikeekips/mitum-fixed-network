// +build test

package node

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (ln *Local) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		AD  base.Address   `json:"address"`
		PUK key.Publickey  `json:"publickey"`
		PRK key.Privatekey `json:"privatekey"`
	}{
		HintedHead: jsonenc.NewHintedHead(ln.Hint()),
		AD:         ln.Address(),
		PUK:        ln.Publickey(),
		PRK:        ln.privatekey,
	})
}
