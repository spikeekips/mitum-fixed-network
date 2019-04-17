package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type testTime struct {
	suite.Suite
}

func (t *testTime) TestNew() {
	nowString := "2019-04-16T16:39:43.665218000+09:00"
	now, _ := ParseISO8601(nowString)

	tn := Time{Time: now}
	t.Equal(nowString, tn.String())
}

func (t *testTime) TestJSON() {
	nowString := "2019-04-16T16:39:43.665218000+09:00"
	now, _ := ParseISO8601(nowString)
	nowTime := Time{Time: now}

	b, err := json.Marshal(nowTime)
	t.NoError(err)

	var returned Time
	err = json.Unmarshal(b, &returned)
	t.NoError(err)

	t.Equal(now, returned.Time)
}

func TestTime(t *testing.T) {
	suite.Run(t, new(testTime))
}
