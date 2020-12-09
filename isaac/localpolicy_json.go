package isaac

import (
	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (lp *LocalPolicy) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		NID string              `json:"network_id"`
		TH  base.ThresholdRatio `json:"threshold"`
		MS  uint                `json:"max_operations_in_seal"`
		MP  uint                `json:"max_operations_in_proposal"`
		TP  string              `json:"timeout_waiting_proposal"`
		II  string              `json:"interval_broadcasting_init_ballot"`
		PR  string              `json:"interval_broadcasting_proposal"`
		WB  string              `json:"wait_broadcasting_accept_ballot"`
		IA  string              `json:"interval_broadcasting_accept_ballot"`
		TS  string              `json:"timespan_valid_ballot"`
		TC  string              `json:"timeout_process_proposal"`
	}{
		NID: string(lp.NetworkID()),
		TH:  lp.ThresholdRatio(),
		MS:  lp.MaxOperationsInSeal(),
		MP:  lp.MaxOperationsInProposal(),
		TP:  lp.TimeoutWaitingProposal().String(),
		II:  lp.IntervalBroadcastingINITBallot().String(),
		PR:  lp.IntervalBroadcastingProposal().String(),
		WB:  lp.WaitBroadcastingACCEPTBallot().String(),
		IA:  lp.IntervalBroadcastingACCEPTBallot().String(),
		TS:  lp.TimespanValidBallot().String(),
		TC:  lp.TimeoutProcessProposal().String(),
	})
}
