// +build test

package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/network"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (ls Local) MarshalJSON() ([]byte, error) {
	var nodes []base.Node
	ls.Nodes().Traverse(func(n base.Node, _ network.Channel) bool {
		nodes = append(nodes, n)
		return true
	})

	return jsonenc.Marshal(struct {
		ND *node.Local  `json:"node"`
		PL *LocalPolicy `json:"policy"`
		NS []base.Node  `json:"nodes"`
	}{
		ND: ls.Node(),
		PL: ls.Policy(),
		NS: nodes,
	})
}
