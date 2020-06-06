package util

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

func (t *testVersionGo) TestLong() {
	v0 := Version("v0.0.1-proto3+commit.449cdb2-patched.ed86a2a70719bef50804b3980f13c68f")
	t.NoError(v0.IsValid(nil))
}

func TestVersionGo(t *testing.T) {
	suite.Run(t, new(testVersionGo))
}
