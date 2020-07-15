package util

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testExec struct {
	suite.Suite
}

func (t *testExec) TestBadCommand() {
	t.Error(Exec("tr0"))
}

func (t *testExec) TestExitCode() {
	t.Error(Exec("exit 1"))
	t.NoError(Exec("exit 0"))
}

func (t *testExec) TestPipe() {
	t.NoError(Exec("echo showme | cat"))
}

func (t *testExec) TestError() {
	t.NoError(Exec("unknown-command || true"))
}

func TestExec(t *testing.T) {
	suite.Run(t, new(testExec))
}
