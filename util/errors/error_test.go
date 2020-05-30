package errors

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testError struct {
	suite.Suite
}

func (t *testError) TestNew() {
	e0 := NewError("showme")
	t.Implements((*(interface{ Error() string }))(nil), e0)

	t.Equal("showme", e0.Error())

	t.True(xerrors.Is(e0, e0))
	t.False(xerrors.Is(e0, NewError("showme")))
	t.False(xerrors.Is(e0, NewError("findme")))
	t.True(xerrors.Is(e0, e0.Errorf("showme")))

	var e1 *NError
	t.True(xerrors.As(e0, &e1))
}

func (t *testError) TestWrap() {
	e0 := NewError("showme")

	pe := &os.PathError{Err: fmt.Errorf("path error")}
	e1 := e0.Wrap(pe)

	t.False(xerrors.Is(e1, NewError("showme")))
	t.True(xerrors.Is(e1, e1.Errorf("showme")))
	t.True(xerrors.Is(e1, pe))

	var e2 *NError
	t.True(xerrors.As(e0, &e2))
	t.True(xerrors.As(e1, &e2))

	var npe *os.PathError
	t.True(xerrors.As(e1, &npe))
}

func (t *testError) TestErrorf() {
	e0 := NewError("showme")
	pe := &os.PathError{Err: fmt.Errorf("path error")}
	e1 := e0.Wrap(pe)

	var e2 *NError
	t.True(xerrors.As(e0, &e2))
	t.True(xerrors.As(e1, &e2))

	t.True(xerrors.Is(e0, e1))
	t.True(xerrors.Is(e1, e1))

	var npe *os.PathError
	t.True(xerrors.As(e1, &npe))
}

func TestError(t *testing.T) {
	suite.Run(t, new(testError))
}
