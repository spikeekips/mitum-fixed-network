package isaac

import (
	"github.com/spikeekips/mitum/util"
)

func (lp LocalPolicy) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		TH Threshold `json:"threshold"`
		TP string    `json:"timeout_waiting_proposal"`
		II string    `json:"interval_broadcasting_init_ballot"`
		WB string    `json:"wait_broadcasting_accept_ballot"`
		IA string    `json:"interval_broadcasting_accept_ballot"`
		NA uint      `json:"number_of_acting_suffrage_nodes"`
	}{
		TH: lp.Threshold(),
		TP: lp.TimeoutWaitingProposal().String(),
		II: lp.IntervalBroadcastingINITBallot().String(),
		WB: lp.WaitBroadcastingACCEPTBallot().String(),
		IA: lp.IntervalBroadcastingACCEPTBallot().String(),
		NA: lp.NumberOfActingSuffrageNodes(),
	})

}

func (ls LocalState) MarshalJSON() ([]byte, error) {
	var nodes []Node
	ls.Nodes().Traverse(func(n Node) bool {
		nodes = append(nodes, n)
		return true
	})

	return util.JSONMarshal(struct {
		ND *LocalNode   `json:"node"`
		PL *LocalPolicy `json:"policy"`
		NS []Node       `json:"nodes"`
		LB Block        `json:"last_block"`
		IV VoteProof    `json:"last_init_voteproof"`
		AV VoteProof    `json:"last_accept_voteproof"`
	}{
		ND: ls.Node(),
		PL: ls.Policy(),
		NS: nodes,
		LB: ls.LastBlock(),
		IV: ls.LastINITVoteProof(),
		AV: ls.LastACCEPTVoteProof(),
	})
}
