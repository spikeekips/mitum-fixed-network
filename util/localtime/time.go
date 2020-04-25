package localtime

import "time"

// ParseTimeFromRFC3339 parses and returns time.Time.
func ParseTimeFromRFC3339(s string) (time.Time, error) {
	// t, err := time.Parse(time.RFC3339Nano, s)
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

// Normalize clear the nanoseconds part from Time and make time to UTC.
func Normalize(t time.Time) time.Time {
	n := t.UTC()

	nsec := n.Nanosecond()

	return time.Date(
		n.Year(),
		n.Month(),
		n.Day(),
		n.Hour(),
		n.Minute(),
		n.Second(),
		(nsec/1000000)*1000000,
		time.UTC,
	)
}

func String(t time.Time) string {
	return RFC3339(t)
}
