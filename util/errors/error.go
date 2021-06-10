package errors

import (
	"fmt"

	uuid "github.com/satori/go.uuid"
	"golang.org/x/xerrors"
)

type NError struct {
	id    string
	s     string
	err   error
	frame xerrors.Frame
}

func NewError(s string, a ...interface{}) *NError {
	var id string
	if u, err := uuid.NewV4(); err != nil {
		panic(err)
	} else {
		id = u.String()
	}

	return &NError{id: id, s: fmt.Sprintf(s, a...)}
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

func (ne *NError) Msg() string {
	return ne.s
}

func (ne *NError) Err() error {
	return ne.err
}

func (ne *NError) Error() string {
	if ne.err == nil {
		return ne.s
	}

	var s string
	if len(ne.s) > 0 {
		s = ne.s + "; "
	}

	return fmt.Sprintf("%s%+v", s, ne.err)
}

func (ne *NError) Is(err error) bool {
	if err == nil {
		return false
	}

	var e *NError
	if !xerrors.As(err, &e) {
		return false
	}
	return e.id == ne.id
}

func (ne *NError) As(target interface{}) bool {
	if ne.err == nil {
		return false
	}

	return xerrors.As(ne.err, target)
}

func (ne *NError) New() *NError {
	n := NewError(ne.s)
	n.id = ne.id

	return n
}

func (ne *NError) Wrap(err error) *NError {
	var nne *NError
	if xerrors.As(err, &nne) {
		if xerrors.Is(err, ne) {
			return nne
		}
	}

	return &NError{
		id:    ne.id,
		s:     ne.s,
		err:   err,
		frame: xerrors.Caller(2),
	}
}

func (ne *NError) Errorf(s string, a ...interface{}) *NError {
	return &NError{
		id:    ne.id,
		s:     ne.s,
		err:   fmt.Errorf(s, a...),
		frame: xerrors.Caller(1),
	}
}

func (ne *NError) SetFrame(n int) *NError {
	ne.frame = xerrors.Caller(n)

	return ne
}
