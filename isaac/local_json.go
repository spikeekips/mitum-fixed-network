package isaac

import (
	"github.com/spikeekips/mitum/network"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (ls Local) MarshalJSON() ([]byte, error) {
	var nodes []network.Node
	ls.Nodes().Traverse(func(n network.Node) bool {
		nodes = append(nodes, n)
		return true
	})

	return jsonenc.Marshal(struct {
		ND *LocalNode     `json:"node"`
		PL *LocalPolicy   `json:"policy"`
		NS []network.Node `json:"nodes"`
	}{
		ND: ls.Node(),
		PL: ls.Policy(),
		NS: nodes,
	})
}
