package localtime

func (t Time) MarshalText() ([]byte, error) {
	return []byte(t.Normalize().RFC3339()), nil
}

func (t *Time) UnmarshalText(b []byte) error {
	s, err := ParseRFC3339(string(b))
	if err != nil {
		return err
	}
	t.Time = Normalize(s)

	return nil
}
