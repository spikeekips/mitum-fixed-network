package localtime

import "time"

// ParseRFC3339 parses RFC3339 string.
func ParseRFC3339(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err != nil {
		return time.Time{}, err
	} else {
		return t, nil
	}
}

// RFC3339 formats time.Time to RFC3339Nano string.
func RFC3339(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}

// Normalize clear the nanoseconds part from Time and make time to UTC.
// "2009-11-10T23:00:00.00101010Z" -> "2009-11-10T23:00:00.001Z",
func Normalize(t time.Time) time.Time {
	n := t.UTC()

	return time.Date(
		n.Year(),
		n.Month(),
		n.Day(),
		n.Hour(),
		n.Minute(),
		n.Second(),
		(n.Nanosecond()/1000000)*1000000,
		time.UTC,
	)
}

func String(t time.Time) string {
	return RFC3339(t)
}

func Equal(a, b time.Time) bool {
	return Normalize(a).Equal(Normalize(b))
}

type Time struct {
	time.Time
}

func NewTime(t time.Time) Time {
	return Time{Time: t}
}

func (t Time) Bytes() []byte {
	return []byte(t.Normalize().String())
}

func (t Time) RFC3339() string {
	return RFC3339(t.Time)
}

func (t Time) Normalize() Time {
	t.Time = Normalize(t.Time)

	return t
}

func (t Time) Equal(n Time) bool {
	return t.Time.Equal(n.Time)
}
