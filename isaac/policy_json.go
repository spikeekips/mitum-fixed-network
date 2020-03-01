package isaac

import "github.com/spikeekips/mitum/util"

func (lp LocalPolicy) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		NID string    `json:"network_id"`
		TH  Threshold `json:"threshold"`
		TP  string    `json:"timeout_waiting_proposal"`
		II  string    `json:"interval_broadcasting_init_ballot"`
		WB  string    `json:"wait_broadcasting_accept_ballot"`
		IA  string    `json:"interval_broadcasting_accept_ballot"`
		NA  uint      `json:"number_of_acting_suffrage_nodes"`
	}{
		NID: string(lp.NetworkID()),
		TP:  lp.TimeoutWaitingProposal().String(),
		II:  lp.IntervalBroadcastingINITBallot().String(),
		WB:  lp.WaitBroadcastingACCEPTBallot().String(),
		IA:  lp.IntervalBroadcastingACCEPTBallot().String(),
		NA:  lp.NumberOfActingSuffrageNodes(),
	})
}
