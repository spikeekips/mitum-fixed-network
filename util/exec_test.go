package util

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type testExec struct {
	suite.Suite
}

func (t *testExec) TestBadCommand() {
	_, err := ShellExec(context.TODO(), "tr0")
	t.Error(err)
}

func (t *testExec) TestExitCode() {
	_, err := ShellExec(context.TODO(), "exit 1")
	t.Error(err)
	_, err = ShellExec(context.TODO(), "exit 0")
	t.NoError(err)
}

func (t *testExec) TestPipe() {
	b, err := ShellExec(context.TODO(), "echo showme | cat")
	t.NoError(err)
	t.Equal([]byte("showme\n"), b)
}

func (t *testExec) TestError() {
	_, err := ShellExec(context.TODO(), "unknown-command || true")
	t.NoError(err)
}

func (t *testExec) TestOutput() {
	// stderr
	b, err := ShellExec(context.TODO(), "echo findme > /dev/stderr")
	t.NoError(err)
	t.Equal([]byte("findme\n"), b)

	// stdout
	b, err = ShellExec(context.TODO(), "echo findme > /dev/stdout")
	t.NoError(err)
	t.Equal([]byte("findme\n"), b)

	// combined
	b, err = ShellExec(context.TODO(), "echo findme > /dev/stdout; echo showme > /dev/stderr")
	t.NoError(err)
	t.Equal([]byte("findme\nshowme\n"), b)
}

func (t *testExec) TestTimeout() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	_, err := ShellExec(ctx, "sleep 4")
	t.Contains(err.Error(), "context deadline exceeded")
}

func TestExec(t *testing.T) {
	suite.Run(t, new(testExec))
}
