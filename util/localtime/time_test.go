package localtime

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testRFC3339 struct {
	suite.Suite
}

func (t *testRFC3339) TestNew() {
	s := "2019-12-26T14:21:00+09:00"
	rf, err := ParseTimeFromRFC3339(s)
	t.NoError(err)
	t.Equal(s, RFC3339(rf))
}

func (t *testRFC3339) TestNewFromExtra() {
	s := "2019-12-26T14:21:00.382503+09:00"
	rf, err := ParseTimeFromRFC3339(s)
	t.NoError(err)
	t.Equal(s, RFC3339(rf))
}

func TestRFC3339(t *testing.T) {
	suite.Run(t, new(testRFC3339))
}
