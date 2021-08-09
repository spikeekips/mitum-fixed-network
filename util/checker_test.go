package util

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
)

type testChecker struct {
	suite.Suite
}

type checkerStruct struct {
	c0keep    bool
	c0err     error
	c1keep    bool
	c1err     error
	c2keep    bool
	c2err     error
	c0checked bool
	c1checked bool
	c2checked bool
}

func (cs *checkerStruct) Check0() (bool, error) {
	defer func() { cs.c0checked = true }()

	return cs.c0keep, cs.c0err
}

func (cs *checkerStruct) Check1() (bool, error) {
	defer func() { cs.c1checked = true }()

	return cs.c1keep, cs.c1err
}

func (cs *checkerStruct) Check2() (bool, error) {
	defer func() { cs.c2checked = true }()

	return cs.c2keep, cs.c2err
}

func (t *testChecker) TestNew() {
	cs := &checkerStruct{
		c0keep: true, c0err: nil,
		c1keep: true, c1err: nil,
		c2keep: true, c2err: nil,
	}

	ck := NewChecker("test", []CheckerFunc{cs.Check0, cs.Check1, cs.Check2})

	t.NoError(ck.Check())
	t.True(cs.c0checked)
	t.True(cs.c1checked)
	t.True(cs.c2checked)
}

func (t *testChecker) TestKeep() {
	cs := &checkerStruct{
		c0keep: true, c0err: nil,
		c1keep: false, c1err: nil,
		c2keep: true, c2err: nil,
	}

	ck := NewChecker("test", []CheckerFunc{cs.Check0, cs.Check1, cs.Check2})

	t.NoError(ck.Check())
	t.True(cs.c0checked)
	t.True(cs.c1checked)
	t.False(cs.c2checked)
}

func (t *testChecker) TestError() {
	cs := &checkerStruct{
		c0keep: true, c0err: nil,
		c1keep: true, c1err: errors.Errorf("show me"),
		c2keep: true, c2err: nil,
	}

	ck := NewChecker("test", []CheckerFunc{cs.Check0, cs.Check1, cs.Check2})

	err := ck.Check()
	t.Contains(err.Error(), "show me")

	t.True(cs.c0checked)
	t.True(cs.c1checked)
	t.False(cs.c2checked)
}

func TestChecker(t *testing.T) {
	defer goleak.VerifyNone(t)

	suite.Run(t, new(testChecker))
}
