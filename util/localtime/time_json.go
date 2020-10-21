package localtime

func (t Time) MarshalText() ([]byte, error) {
	return []byte(t.Normalize().RFC3339()), nil
}

func (t *Time) UnmarshalText(b []byte) error {
	if s, err := ParseRFC3339(string(b)); err != nil {
		return err
	} else {
		t.Time = Normalize(s)

		return nil
	}
}
