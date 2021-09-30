package network

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type NilConnInfoPackerJSON struct {
	jsonenc.HintedHead
	S string `json:"name"`
}

func (conn NilConnInfo) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(NilConnInfoPackerJSON{
		HintedHead: jsonenc.NewHintedHead(conn.Hint()),
		S:          conn.s,
	})
}

type NilConnInfoUnpackerJSON struct {
	S string `json:"name"`
}

func (conn *NilConnInfo) UnmarshalJSON(b []byte) error {
	var uc NilConnInfoUnpackerJSON
	if err := jsonenc.Unmarshal(b, &uc); err != nil {
		return err
	}

	conn.s = uc.S

	return nil
}

type HTTPConnInfoPackerJSON struct {
	jsonenc.HintedHead
	U string `json:"url"`
	I bool   `json:"insecure"`
}

func (conn HTTPConnInfo) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(HTTPConnInfoPackerJSON{
		HintedHead: jsonenc.NewHintedHead(conn.Hint()),
		U:          conn.u.String(),
		I:          conn.insecure,
	})
}

type HTTPConnInfoUnpackerJSON struct {
	U string `json:"url"`
	I bool   `json:"insecure"`
}

func (conn *HTTPConnInfo) UnmarshalJSON(b []byte) error {
	var uc HTTPConnInfoUnpackerJSON
	if err := jsonenc.Unmarshal(b, &uc); err != nil {
		return err
	}

	return conn.unpack(uc.U, uc.I)
}
