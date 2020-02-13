package isaac

import (
	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/util"
)

type VoteProofGenesisV0PackJSON struct {
	encoder.JSONPackHintedHead
	HT Height              `json:"height"`
	RD Round               `json:"round"`
	TH Threshold           `json:"threshold"`
	RS VoteProofResultType `json:"result"`
	ST Stage               `json:"stage"`
	MJ interface{}         `json:"majority"`
	FS interface{}         `json:"facts"`
	BS interface{}         `json:"ballots"`
	VS interface{}         `json:"votes"`
}

func (vpg VoteProofGenesisV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(VoteProofGenesisV0PackJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(vpg.Hint()),
		HT:                 vpg.height,
		RD:                 vpg.Round(),
		TH:                 vpg.threshold,
		RS:                 vpg.Result(),
		ST:                 vpg.stage,
	})
}

type VoteProofGenesisV0UnpackJSON struct {
	HT Height    `json:"height"`
	TH Threshold `json:"threshold"`
	ST Stage     `json:"stage"`
}

func (vpg *VoteProofGenesisV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error { // nolint
	var vpp VoteProofGenesisV0UnpackJSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	vpg.height = vpp.HT
	vpg.threshold = vpp.TH
	vpg.stage = vpp.ST

	return nil
}
