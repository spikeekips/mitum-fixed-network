package util

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

var (
	IgnoreError     = NewError("ignore")
	NotFoundError   = NewError("not found")
	FoundError      = NewError("found")
	DuplicatedError = NewError("duplicated error")
	WrongTypeError  = NewError("wrong type")
	EmptyError      = NewError("empty")
)

type DataContainerError struct {
	d interface{}
}

func NewDataContainerError(d interface{}) DataContainerError {
	return DataContainerError{d: d}
}

func (er DataContainerError) Error() string {
	return fmt.Sprintf("data container error of %T", er.d)
}

func (er DataContainerError) Data() interface{} {
	return er.d
}

func EnsureErrors(
	ctx context.Context,
	dur time.Duration,
	f func() error,
	errs ...error,
) error {
	var wait func()
	if dur > 0 {
		wait = func() {
			<-time.After(dur)
		}
	}

	var err error
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err = f()
			if err == nil {
				return nil
			}

			var found bool
			for i := range errs {
				if errors.Is(err, errs[i]) {
					found = true

					break
				}
			}

			if !found {
				return err
			}
		}

		wait()
	}
}
