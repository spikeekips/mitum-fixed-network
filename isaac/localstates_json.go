package isaac

import (
	"github.com/spikeekips/mitum/network"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

func (ls Localstate) MarshalJSON() ([]byte, error) {
	var nodes []network.Node
	ls.Nodes().Traverse(func(n network.Node) bool {
		nodes = append(nodes, n)
		return true
	})

	return jsonencoder.Marshal(struct {
		ND *LocalNode     `json:"node"`
		PL *LocalPolicy   `json:"policy"`
		NS []network.Node `json:"nodes"`
	}{
		ND: ls.Node(),
		PL: ls.Policy(),
		NS: nodes,
	})
}
