package localtime

import "time"

// ParseTimeFromRFC3339 parses and returns time.Time.
func ParseTimeFromRFC3339(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

// RFC3339 formats time.Time to RFC3339Nano string.
func RFC3339(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}
