package isaac

import (
	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/util"
)

type VoteproofGenesisV0PackJSON struct {
	encoder.JSONPackHintedHead
	HT Height              `json:"height"`
	RD Round               `json:"round"`
	TH Threshold           `json:"threshold"`
	RS VoteproofResultType `json:"result"`
	ST Stage               `json:"stage"`
	MJ interface{}         `json:"majority"`
	FS interface{}         `json:"facts"`
	BS interface{}         `json:"ballots"`
	VS interface{}         `json:"votes"`
}

func (vpg VoteproofGenesisV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(VoteproofGenesisV0PackJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(vpg.Hint()),
		HT:                 vpg.height,
		RD:                 vpg.Round(),
		TH:                 vpg.threshold,
		RS:                 vpg.Result(),
		ST:                 vpg.stage,
	})
}

type VoteproofGenesisV0UnpackJSON struct {
	HT Height    `json:"height"`
	TH Threshold `json:"threshold"`
	ST Stage     `json:"stage"`
}

func (vpg *VoteproofGenesisV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error { // nolint
	var vpp VoteproofGenesisV0UnpackJSON
	if err := enc.Unmarshal(b, &vpp); err != nil {
		return err
	}

	vpg.height = vpp.HT
	vpg.threshold = vpp.TH
	vpg.stage = vpp.ST

	return nil
}
