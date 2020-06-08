package localtime

import (
	"time"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type JSONTime struct {
	time.Time
}

func NewJSONTime(t time.Time) JSONTime {
	return JSONTime{Time: t}
}

func (jt JSONTime) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(RFC3339(jt.Time))
}

func (jt *JSONTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := jsonenc.Unmarshal(b, &s); err != nil {
		return err
	}

	t, err := ParseTimeFromRFC3339(s)
	if err != nil {
		return err
	}
	jt.Time = t

	return nil
}
