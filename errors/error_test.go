package errors

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testError struct {
	suite.Suite
}

func (t *testError) TestNew() {
	error0 := NewError("0 error")
	t.True(error0.Is(error0))
	t.True(xerrors.Is(error0, error0))

	error01 := error0.Wrapf("findme")
	t.True(error0.Is(error01))
	t.True(error01.(CError).Is(error0))
	t.True(xerrors.Is(error01, error0))
}

func (t *testError) TestAs0() {
	error0 := NewError("0 error")

	var error01 CError
	t.True(xerrors.As(error0, &error01))
}

func (t *testError) TestAs1() {
	error0 := NewError("0 error")
	error1 := error0.Wrap(os.ErrClosed)

	t.Equal(os.ErrClosed, error1.(CError).Unwrap())

	var error2 error
	t.True(xerrors.As(error1, &error2))
}

func (t *testError) TestIs() {
	error0 := NewError("0 error")
	error1 := error0.Wrap(os.ErrClosed)

	t.True(xerrors.Is(error1, error0))
	t.True(xerrors.Is(error1, os.ErrClosed))
	t.False(xerrors.Is(error1, os.ErrNotExist))
}

func TestError(t *testing.T) {
	suite.Run(t, new(testError))
}
