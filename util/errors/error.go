package errors

import (
	"fmt"

	"golang.org/x/xerrors"
)

type NError struct {
	s     string
	err   error
	frame xerrors.Frame
}

func NewError(s string, a ...interface{}) *NError {
	return &NError{s: fmt.Sprintf(s, a...)}
}

func (ne *NError) Unwrap() error {
	return ne.err
}

func (ne *NError) Format(s fmt.State, v rune) {
	xerrors.FormatError(ne, s, v)
}

func (ne *NError) FormatError(p xerrors.Printer) error {
	p.Print(ne.s)
	ne.frame.Format(p)

	return ne.err
}

func (ne *NError) Error() string {
	if ne.err == nil {
		return ne.s
	}

	return fmt.Sprintf("%s; %+v", ne.s, ne.err)
}

func (ne *NError) Is(err error) bool {
	if err == nil {
		return false
	}

	if e, ok := err.(*NError); !ok {
		return false
	} else {
		return e.s == ne.s
	}
}

func (ne *NError) New() *NError {
	return NewError(ne.s)
}

func (ne *NError) Wrap(err error) *NError {
	return &NError{
		s:     ne.s,
		err:   err,
		frame: xerrors.Caller(2),
	}
}

func (ne *NError) Errorf(s string, a ...interface{}) *NError {
	return &NError{
		s:     ne.s,
		err:   NewError(fmt.Sprintf("%s; %s", ne.s, s), a...).SetFrame(2),
		frame: xerrors.Caller(2),
	}
}

func (ne *NError) SetFrame(n int) *NError {
	ne.frame = xerrors.Caller(n)

	return ne
}
