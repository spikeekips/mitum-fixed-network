package network

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type ProblemJSONPacker struct {
	jsonenc.HintedHead
	T  string                 `json:"type"`
	TI string                 `json:"title"`
	DE string                 `json:"detail,omitempty"`
	EX map[string]interface{} `json:"extra,omitempty"`
}

func (pr Problem) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(ProblemJSONPacker{
		HintedHead: jsonenc.NewHintedHead(pr.Hint()),
		T:          pr.t,
		TI:         pr.title,
		DE:         pr.detail,
		EX:         pr.extra,
	})
}

func (pr *Problem) UnmarshalJSON(b []byte) error {
	var upr ProblemJSONPacker
	if err := jsonenc.Unmarshal(b, &upr); err != nil {
		return err
	}

	pr.t = upr.T
	pr.title = upr.TI
	pr.detail = upr.DE
	pr.extra = upr.EX

	return nil
}
