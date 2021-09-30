package network

func (conn *HTTPConnInfo) unpack(u string, insecure bool) error {
	uu, err := ParseURL(u, true)
	if err != nil {
		return err
	}

	conn.u = uu
	conn.insecure = insecure

	return nil
}
