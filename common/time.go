package common

import (
	"encoding/json"
	"time"
)

const (
	TIMEFORMAT_ISO8601 string = "2006-01-02T15:04:05.000000000Z07:00"
)

func FormatISO8601(t time.Time) string {
	return t.Format(TIMEFORMAT_ISO8601)
}

func NowISO8601() string {
	return FormatISO8601(time.Now())
}

func ParseISO8601(s string) (time.Time, error) {
	return time.Parse(TIMEFORMAT_ISO8601, s)
}

type Time struct {
	time.Time
}

func (t Time) String() string {
	return FormatISO8601(t.Time)
}

func (t Time) MarshalText() ([]byte, error) {
	return json.Marshal(FormatISO8601(t.Time))
}

func (t *Time) UnmarshalText(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	n, err := ParseISO8601(s)
	if err != nil {
		return err
	}

	*t = Time{Time: n}

	return nil
}

func Now() Time {
	return Time{Time: time.Now()}
}
