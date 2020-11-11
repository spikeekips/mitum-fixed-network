package policy

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type PolicyV0PackerJSON struct {
	jsonenc.HintedHead
	NS uint `json:"number_of_acting_suffrage_nodes"`
	MS uint `json:"max_operations_in_seal"`
	MP uint `json:"max_operations_in_proposal"`
}

func (po PolicyV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(PolicyV0PackerJSON{
		HintedHead: jsonenc.NewHintedHead(po.Hint()),
		NS:         po.numberOfActingSuffrageNodes,
		MS:         po.maxOperationsInSeal,
		MP:         po.maxOperationsInProposal,
	})
}

func (po *PolicyV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var up PolicyV0PackerJSON
	if err := enc.Unmarshal(b, &up); err != nil {
		return err
	}

	return po.unpack(up.NS, up.MS, up.MP)
}
