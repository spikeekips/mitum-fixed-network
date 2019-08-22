package common

import (
	"context"
	"reflect"
	"sync"

	"golang.org/x/xerrors"
)

const (
	ChainCheckerStopErrorCode ErrorCode = iota + 1
	ContextValueNotFoundErrorCode
)

var (
	ChainCheckerStopError     = NewError("chain", ChainCheckerStopErrorCode, "chain stopped")
	ContextValueNotFoundError = NewError("chain", ContextValueNotFoundErrorCode, "value not found in context")
)

type ChainCheckerFunc func(*ChainChecker) error

type ChainChecker struct {
	sync.RWMutex
	*Logger
	checkers    []ChainCheckerFunc
	originalCtx context.Context
	ctx         context.Context
}

func NewChainChecker(name string, ctx context.Context, checkers ...ChainCheckerFunc) *ChainChecker {
	return &ChainChecker{
		Logger:      NewLogger(log, "module", name),
		checkers:    checkers,
		ctx:         ctx,
		originalCtx: ctx,
	}
}

func (c *ChainChecker) New(ctx context.Context) *ChainChecker {
	c.RLock()
	defer c.RUnlock()

	if ctx == nil || ctx == context.TODO() {
		ctx = c.originalCtx
	}

	return &ChainChecker{
		Logger:      c.Logger,
		checkers:    c.checkers,
		ctx:         ctx,
		originalCtx: ctx,
	}
}

func (c *ChainChecker) Error() string {
	return "ChainChecker will be also chained"
}

func (c *ChainChecker) Context() context.Context {
	c.RLock()
	defer c.RUnlock()

	return c.ctx
}

func (c *ChainChecker) SetContext(key, value interface{}) *ChainChecker {
	c.Lock()
	defer c.Unlock()

	c.ctx = SetContext(c.ctx, key, value)

	return c
}

func (c *ChainChecker) ContextValue(key interface{}, value interface{}) error {
	v := c.Context().Value(key)
	if v == nil {
		return ContextValueNotFoundError.Newf("key='%v'", key)
	}

	reflect.ValueOf(value).Elem().Set(reflect.ValueOf(v))

	return nil
}

func (c *ChainChecker) Check() error {
	// initialize context
	c.Lock()
	c.originalCtx = c.ctx
	c.ctx = c.originalCtx
	c.Unlock()

	var err error
	var newChecker *ChainChecker

end:
	for _, f := range c.checkers {
		err = f(c)

		if err == nil {
			continue
		}

		switch err := err.(type) {
		case *ChainChecker:
			newChecker = err
			break end
		default:
			if xerrors.Is(err, ChainCheckerStopError) {
				c.Log().Debug("checker stopped", "stop", err)
				return nil
			}

			return err
		}
	}

	if newChecker == nil {
		return nil
	}

	newChecker.SetLogContext(c.LogContext())
	err = newChecker.Check()

	c.Lock()
	c.ctx = newChecker.Context()
	c.Unlock()

	return err
}
