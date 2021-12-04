package network

import "github.com/spikeekips/mitum/util/hint"

func (conn *HTTPConnInfo) unpack(ht hint.Hint, u string, insecure bool) error {
	uu, err := ParseURL(u, true)
	if err != nil {
		return err
	}

	conn.BaseHinter = hint.NewBaseHinter(ht)
	conn.u = uu
	conn.insecure = insecure

	return nil
}
