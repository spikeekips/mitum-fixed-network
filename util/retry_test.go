package util

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
)

type testRetry struct {
	suite.Suite
}

func (t *testRetry) TestNoError() {
	var called int
	err := Retry(3, 0, func(int) error {
		called++

		return nil
	})
	t.NoError(err)
	t.Equal(1, called)
}

func (t *testRetry) TestErrorAndSuccess() {
	var called int
	err := Retry(3, 0, func(i int) error {
		called = i
		if i == 1 {
			return errors.Errorf("error")
		}

		return nil
	})
	t.NoError(err)
	t.Equal(0, called)
}

func (t *testRetry) TestError() {
	var called int
	err := Retry(3, 0, func(i int) error {
		called = i

		return errors.Errorf("error: %d", called+1)
	})
	t.Contains(err.Error(), "3")
	t.Equal(2, called)
}

func (t *testRetry) TestStopRetrying() {
	var called int
	_ = Retry(3, 0, func(i int) error {
		called = i
		if called == 1 {
			return StopRetryingError
		}

		return errors.Errorf("error: %d", called+1)
	})
	t.Equal(1, called)
}

func TestRetry(t *testing.T) {
	defer goleak.VerifyNone(t)

	suite.Run(t, new(testRetry))
}
