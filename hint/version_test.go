package hint

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testVersionGo struct {
	suite.Suite
}

func (t *testVersionGo) TestWithoutPrefix() {
	v0 := "0.1.1"
	t.Equal("v"+v0, VersionGO(v0))
}

func (t *testVersionGo) TestWithPrefix() {
	v0 := "v0.1.1"
	t.Equal(v0, VersionGO(v0))
}

func (t *testVersionGo) TestWithMultiplePrefix() {
	v0 := "vv0.1.1"
	t.Equal(v0, VersionGO(v0))
}

func TestVersionGo(t *testing.T) {
	suite.Run(t, new(testVersionGo))
}
