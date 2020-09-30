package cache

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testDummy struct {
	suite.Suite
}

func (t *testDummy) TestNew() {
	ca := Dummy{}

	_, ok := (interface{})(ca).(Cache)
	t.True(ok)
}

func TestDummy(t *testing.T) {
	suite.Run(t, new(testDummy))
}
