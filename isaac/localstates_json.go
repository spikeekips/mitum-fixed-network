package isaac

import (
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

func (ls Localstate) MarshalJSON() ([]byte, error) {
	var nodes []Node
	ls.Nodes().Traverse(func(n Node) bool {
		nodes = append(nodes, n)
		return true
	})

	return jsonencoder.Marshal(struct {
		ND *LocalNode   `json:"node"`
		PL *LocalPolicy `json:"policy"`
		NS []Node       `json:"nodes"`
	}{
		ND: ls.Node(),
		PL: ls.Policy(),
		NS: nodes,
	})
}
