package hint

func (hs HintedString) MarshalText() ([]byte, error) {
	return []byte(hs.String()), nil
}

func (hs *HintedString) UnmarshalText(b []byte) error {
	if len(b) < 1 {
		return nil
	}

	i, err := ParseHintedString(string(b))
	if err != nil {
		return err
	}

	hs.h = i.h
	hs.s = i.s

	return nil
}

func (ts TypedString) MarshalText() ([]byte, error) {
	return []byte(ts.String()), nil
}

func (ts *TypedString) UnmarshalText(b []byte) error {
	if len(b) < 1 {
		return nil
	}

	i, err := ParseTypedString(string(b))
	if err != nil {
		return err
	}

	ts.t = i.t
	ts.s = i.s

	return nil
}
