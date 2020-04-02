package errors

import (
	"fmt"

	"golang.org/x/xerrors"
)

// CError is simple wrapper for xerror.Is and xerror.As.
type CError struct {
	S     string
	Err   error
	Frame xerrors.Frame
}

func NewError(s string, a ...interface{}) CError {
	return CError{S: fmt.Sprintf(s, a...)}
}

// TODO something wrong, needs rewriting

// Wrap put error inside Error.
func (we CError) Wrap(err error) error {
	return CError{
		S:     we.S,
		Err:   err,
		Frame: xerrors.Caller(1),
	}
}

// Wrapf acts like `fmt.Errorf()`.
func (we CError) Wrapf(s string, a ...interface{}) error {
	return CError{
		S:     we.S,
		Err:   xerrors.Errorf(s, a...),
		Frame: xerrors.Caller(1),
	}
}

// Is is for `xerrors.Is()`.
func (we CError) Is(err error) bool {
	if err == nil {
		return false
	}

	e, ok := err.(CError)
	if !ok {
		return false
	}

	return e.S == we.S
}

func (we CError) Unwrap() error {
	return we.Err
}

func (we CError) FormatError(p xerrors.Printer) error {
	we.Frame.Format(p)
	return we.Unwrap()
}

func (we CError) Error() string {
	if we.Err == nil {
		return we.S
	}

	return fmt.Sprintf("%s; %+v", we.S, we.Err)
}
