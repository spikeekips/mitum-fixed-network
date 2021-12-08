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
