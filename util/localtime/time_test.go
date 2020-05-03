package localtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
)

type testTime struct {
	suite.Suite
}

func (t *testTime) TestNormalize() {
	tn := time.Now()

	n := Normalize(tn)

	t.Equal(time.UTC, n.Location())
	t.Equal((tn.Nanosecond()/1000000)*1000000, n.Nanosecond())
}

func TestTime(t *testing.T) {
	defer goleak.VerifyNone(t)

	suite.Run(t, new(testTime))
}
