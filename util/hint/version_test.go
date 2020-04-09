package hint

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testVersionGo struct {
	suite.Suite
}

func (t *testVersionGo) TestWithoutPrefix() {
	v0 := Version("0.1.1")
	t.Equal("v"+v0.String(), v0.GO())
}

func (t *testVersionGo) TestWithPrefix() {
	v0 := Version("v0.1.1")
	t.Equal(v0.String(), v0.GO())
}

func (t *testVersionGo) TestWithMultiplePrefix() {
	v0 := Version("vv0.1.1")
	t.Equal(v0.String(), v0.GO())
}

func TestVersionGo(t *testing.T) {
	suite.Run(t, new(testVersionGo))
}
