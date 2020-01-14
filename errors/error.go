package errors

import (
	"fmt"

	"golang.org/x/xerrors"
)

// Error is simple wrapper for xerror.Is and xerror.As.
type Error struct {
	S     string
	Err   error
	Frame xerrors.Frame
}

func NewError(s string, a ...interface{}) Error {
	return Error{S: fmt.Sprintf(s, a...)}
}

// Wrap put error inside Error.
func (we Error) Wrap(err error) error {
	return Error{
		S:     we.S,
		Err:   err,
		Frame: xerrors.Caller(1),
	}
}

// Wrapf acts like `fmt.Errorf()`.
func (we Error) Wrapf(s string, a ...interface{}) error {
	return Error{
		S:     we.S,
		Err:   xerrors.Errorf(s, a...),
		Frame: xerrors.Caller(1),
	}
}

// Is is for `xerrors.Is()`.
func (we Error) Is(err error) bool {
	if err == nil {
		return false
	}

	e, ok := err.(Error)
	if !ok {
		return false
	}

	return e.S == we.S
}

func (we Error) Unwrap() error {
	return we.Err
}

func (we Error) FormatError(p xerrors.Printer) error {
	we.Frame.Format(p)
	return we.Unwrap()
}

func (we Error) Error() string {
	if we.Err == nil {
		return we.S
	}

	return fmt.Sprintf("%s; %v", we.S, we.Err)
}
