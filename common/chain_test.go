package common

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

type testChain struct {
	suite.Suite
}

func (t *testChain) TestNew() {
	var c1st, c2nd, c3rd, c4th bool
	checker := NewChainChecker(
		"showme-checker",
		context.Background(),
		func(*ChainChecker) error {
			c1st = true
			return nil
		},
		func(*ChainChecker) error {
			c2nd = true
			return nil
		},
		func(*ChainChecker) error {
			c3rd = true
			return nil
		},
		func(*ChainChecker) error {
			c4th = true
			return nil
		},
	)

	err := checker.Check()
	t.NoError(err)

	t.True(c1st)
	t.True(c2nd)
	t.True(c3rd)
	t.True(c4th)
}

func (t *testChain) TestContext() {
	ctx := context.Background()
	checker := NewChainChecker(
		"showme-checker",
		ctx,
		func(c *ChainChecker) error {
			c.SetContext("1st", true)
			return nil
		},
		func(c *ChainChecker) error {
			c.SetContext("2nd", true)
			return nil
		},
	)

	err := checker.Check()
	t.NoError(err)

	t.Equal(true, checker.Context().Value("1st"))
	t.Equal(true, checker.Context().Value("2nd"))
}

func (t *testChain) TestStop() {
	ctx := context.Background()
	checker := NewChainChecker(
		"showme-checker",
		ctx,
		func(c *ChainChecker) error {
			c.SetContext("1st", true)
			return ChainCheckerStopError
		},
		func(c *ChainChecker) error {
			c.SetContext("2nd", true)
			return nil
		},
	)

	err := checker.Check()
	t.NoError(err)

	t.Equal(true, checker.Context().Value("1st"))
	t.Nil(checker.Context().Value("2nd"))
}

func (t *testChain) TestStopError() {
	ctx := context.Background()
	checker := NewChainChecker(
		"showme-checker",
		ctx,
		func(c *ChainChecker) error {
			c.SetContext("1st", true)
			return errors.New("something wrong")
		},
		func(c *ChainChecker) error {
			c.SetContext("2nd", true)
			return nil
		},
	)

	err := checker.Check()
	t.Error(err, "something wrong")

	t.Equal(true, checker.Context().Value("1st"))
	t.Nil(checker.Context().Value("2nd"))
}

func (t *testChain) TestChainCheckerChained() {
	ctx := context.Background()
	checker := NewChainChecker(
		"showme-checker",
		ctx,
		func(c *ChainChecker) error {
			c.SetContext("1st", true)

			chained := NewChainChecker(
				"chained-checker",
				c.Context(),
				func(c *ChainChecker) error {
					c.SetContext("3rd", true)
					return nil
				},
			)

			return chained
		},
		func(c *ChainChecker) error {
			c.SetContext("2nd", true)
			return nil
		},
	)

	err := checker.Check()
	t.NoError(err)

	t.Equal(true, checker.Context().Value("1st"))
	t.Nil(checker.Context().Value("2nd"))
	t.Equal(true, checker.Context().Value("3rd"))
}

func TestChain(t *testing.T) {
	suite.Run(t, new(testChain))
}
