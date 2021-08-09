package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/stretchr/testify/suite"
)

type testError struct {
	suite.Suite
}

func (t *testError) TestIs() {
	e0 := NewError("showme")
	t.Implements((*(interface{ Error() string }))(nil), e0)

	t.Equal("showme", e0.Error())

	t.True(errors.Is(e0, e0))
	t.False(errors.Is(e0, NewError("showme")))
	t.False(errors.Is(e0, NewError("findme")))
	t.True(errors.Is(e0, e0.Errorf("showme")))
}

func (t *testError) TestAs() {
	e0 := NewError("showme")

	var e1 *NError
	t.True(errors.As(e0, &e1))

	t.True(errors.Is(e0, e1))
	t.True(errors.Is(e1, e0))
}

func (t *testError) TestWrap() {
	e0 := NewError("showme")

	pe := &os.PathError{Err: errors.Errorf("path error")}
	e1 := e0.Wrap(pe)

	t.False(errors.Is(e1, NewError("showme")))
	t.True(errors.Is(e1, e1.Errorf("showme")))
	t.True(errors.Is(e1, pe))

	var e2 *NError
	t.True(errors.As(e0, &e2))
	t.True(errors.As(e1, &e2))

	t.True(errors.Is(e0, e2))
	t.True(errors.Is(e1, e2))

	var npe *os.PathError
	t.True(errors.As(e1, &npe))

	t.True(errors.Is(pe, npe))
}

func (t *testError) TestErrorf() {
	e0 := NewError("showme")
	pe := &os.PathError{Err: errors.Errorf("path error")}
	e1 := e0.Errorf("error: %w", pe)

	var e2 *NError
	t.True(errors.As(e0, &e2))
	t.True(errors.As(e1, &e2))

	t.True(errors.Is(e0, e1))
	t.True(errors.Is(e1, e1))

	var npe *os.PathError
	t.True(errors.As(e1, &npe))
}

func (t *testError) printStack(err error) (string, bool) {
	i, ok := err.(interface {
		StackTrace() errors.StackTrace
	})
	if !ok {
		return "<no StackTrace()>", false
	}

	buf := bytes.NewBuffer(nil)

	for _, f := range i.StackTrace() {
		_, _ = fmt.Fprintf(buf, "%+s:%d\n", f, f)
	}

	return buf.String(), true
}

func (t *testError) printStacks(err error) string {
	buf := bytes.NewBuffer(nil)

	_, _ = fmt.Fprintln(buf, "================================================================================")

	var e error = err
	for {
		i, ok := t.printStack(e)
		if ok {
			_, _ = fmt.Fprintln(buf, i)
			_, _ = fmt.Fprintln(buf, "================================================================================")
		}
		e = errors.Unwrap(e)
		if e == nil {
			break
		}
	}

	return buf.String()
}

func (t *testError) TestMerge() {
	e := NewError("showme")
	e0 := errors.New("findme")

	we := e.Wrap(e0)

	t.T().Logf("wrapped,  v: %v", we)
	t.T().Logf("wrapped, +v: %+v", we)

	me := e.Merge(e0)

	t.T().Logf("merged,  v: %v", me)
	t.T().Logf("merged, +v: %+v", me)
}

func (t *testError) TestPrint() {
	e0 := NewError("showme")

	t.T().Logf("e0,  v: %v", e0)
	t.T().Logf("e0, +v: %+v", e0)

	e1 := e0.Errorf("error: %w", &os.PathError{Op: "op", Path: "/tmp", Err: errors.Errorf("path error")})
	t.T().Logf("e1,  v: %v", e1)
	t.T().Logf("e1, +v: %+v", e1)

	e2 := e0.Wrap(&os.PathError{Op: "e2", Path: "/tmp/e2", Err: errors.Errorf("path error")})
	t.T().Logf("e2,  v: %v", e2)
	t.T().Logf("e2, +v: %+v", e2)
}

func (t *testError) TestPrintStacks() {
	e0 := NewError("showme")
	e1 := errors.New("findme")

	e2 := e0.Wrap(e1)
	t.T().Logf("e2,      v: %v", e2)
	t.T().Logf("e2,     +v: %+v", e2)
	t.T().Logf("e2, stacks:\n%s", t.printStacks(e2))
}

func (t *testError) checkStack(b []byte) bool {
	var m map[string]interface{}
	t.NoError(json.Unmarshal(b, &m))

	i := m["stack"]
	stacks, ok := i.([]interface{})
	t.True(ok)
	t.NotNil(stacks)

	var goexitfound bool
	for i := range stacks {
		s := stacks[i]
		sm := s.(map[string]interface{})
		j := sm[pkgerrors.StackSourceFileName]
		k, ok := j.(string)
		t.True(ok)

		goexitfound = k == "goexit"
		if goexitfound {
			break
		}
	}

	return goexitfound
}

func (t *testError) TestPKGErrorStack() {
	e := errors.Errorf("showme")

	var bf bytes.Buffer
	l := logging.Setup(&bf, zerolog.DebugLevel, "json", false)

	l.Log().Error().Err(e).Msg("find")
	t.T().Log(bf.String())

	t.False(t.checkStack(bf.Bytes()))
}

func (t *testError) TestNError() {
	e := NewError("showme").Caller(3)

	var bf bytes.Buffer
	l := logging.Setup(&bf, zerolog.DebugLevel, "json", false)

	l.Log().Error().Err(e).Msg("find")
	t.T().Log(bf.String())

	t.False(t.checkStack(bf.Bytes()))
}

func TestError(t *testing.T) {
	suite.Run(t, new(testError))
}
