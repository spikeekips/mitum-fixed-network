package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

func (ls Localstate) MarshalJSON() ([]byte, error) {
	var nodes []Node
	ls.Nodes().Traverse(func(n Node) bool {
		nodes = append(nodes, n)
		return true
	})

	return jsonencoder.Marshal(struct {
		ND *LocalNode     `json:"node"`
		PL *LocalPolicy   `json:"policy"`
		NS []Node         `json:"nodes"`
		LB block.Block    `json:"last_block"`
		IV base.Voteproof `json:"last_init_voteproof"`
		AV base.Voteproof `json:"last_accept_voteproof"`
	}{
		ND: ls.Node(),
		PL: ls.Policy(),
		NS: nodes,
		LB: ls.LastBlock(),
		IV: ls.LastINITVoteproof(),
		AV: ls.LastACCEPTVoteproof(),
	})
}
