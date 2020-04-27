package isaac

import (
	"github.com/spikeekips/mitum/base"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

func (lp LocalPolicy) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(struct {
		NID string         `json:"network_id"`
		TH  base.Threshold `json:"threshold"`
		TP  string         `json:"timeout_waiting_proposal"`
		II  string         `json:"interval_broadcasting_init_ballot"`
		PR  string         `json:"interval_broadcasting_proposal"`
		WB  string         `json:"wait_broadcasting_accept_ballot"`
		IA  string         `json:"interval_broadcasting_accept_ballot"`
		NA  uint           `json:"number_of_acting_suffrage_nodes"`
		TS  string         `json:"timespan_valid_ballot"`
	}{
		NID: string(lp.NetworkID()),
		TP:  lp.TimeoutWaitingProposal().String(),
		II:  lp.IntervalBroadcastingINITBallot().String(),
		PR:  lp.IntervalBroadcastingProposal().String(),
		WB:  lp.WaitBroadcastingACCEPTBallot().String(),
		IA:  lp.IntervalBroadcastingACCEPTBallot().String(),
		NA:  lp.NumberOfActingSuffrageNodes(),
		TS:  lp.TimespanValidBallot().String(),
	})
}
