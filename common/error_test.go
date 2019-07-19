package common

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testError struct {
	suite.Suite
}

func (t *testError) TestNew() {
	et := NewError("test", 1, "show me")

	_, ok := interface{}(et).(error)
	t.True(ok)
}

func (t *testError) TestEqual() {
	et0 := NewError("test", 1, "show me")
	e0 := et0.New(nil)
	t.True(xerrors.Is(e0, e0.New(nil)))

	et1 := NewError("test", 2, "show me")
	t.False(xerrors.Is(e0, et1.New(nil)))

	t.False(xerrors.Is(e0, errors.New("show me")))
}

func (t *testError) TestWrap() {
	et0 := NewError("test", 1, "show me")

	err := xerrors.Errorf("find me")
	e0 := et0.New(err)

	t.True(xerrors.Is(e0, e0.New(nil)))
	t.True(xerrors.Is(e0, err))
}

func (t *testError) TestNestedWrap() {
	et0 := NewError("test", 1, "show me")
	et1 := NewError("test", 2, "findme me")

	err := xerrors.Errorf("find me")
	e0 := et0.New(err)
	e1 := et1.New(e0)

	{ // Is
		t.False(xerrors.Is(e0, e1))
		t.True(xerrors.Is(e0, err))
		t.False(xerrors.Is(e0, e1))
		t.True(xerrors.Is(e1, e0))
	}

	{ // As: error
		var e2 error
		t.True(xerrors.As(e0, &e2))
		t.True(xerrors.Is(e0, e2))
		t.True(xerrors.Is(e2, e0))
	}

	{ // As: Error
		var e2 Error
		t.True(xerrors.As(e0, &e2))
		t.True(xerrors.Is(e0, e2))
		t.True(xerrors.Is(e2, e0))
	}
}

func TestError(t *testing.T) {
	suite.Run(t, new(testError))
}
