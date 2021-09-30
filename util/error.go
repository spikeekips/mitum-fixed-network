package util

import (
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

type NError struct {
	*stack
	id      string
	msg     string
	wrapped error
}

func NewError(s string, a ...interface{}) *NError {
	var pcs [1]uintptr
	_ = runtime.Callers(2, pcs[:])
	f := errors.Frame(pcs[0])

	return &NError{
		id:  fmt.Sprintf("%n:%d", f, f),
		msg: strings.TrimSpace(fmt.Sprintf(s, a...)),
	}
}

func (er *NError) Error() string {
	i := er.msg

	if er.wrapped != nil {
		j := er.wrapped.Error()
		if len(j) > 0 {
			i += "; " + j
		}
	}

	return i
}

func (er *NError) Unwrap() error {
	return er.wrapped
}

func (er *NError) Is(err error) bool {
	i, ok := err.(*NError) // nolint:errorlint
	if !ok {
		return false
	}

	return i.id == er.id
}

func (*NError) As(err interface{}) bool { // nolint:govet
	_, ok := err.(*NError) // nolint:errorlint

	return ok
}

func (er *NError) Wrap(err error) *NError {
	return &NError{
		id:      er.id,
		msg:     er.msg,
		stack:   callers(3),
		wrapped: err,
	}
}

func (er *NError) Errorf(s string, a ...interface{}) *NError {
	return &NError{
		id:      er.id,
		msg:     er.msg,
		stack:   callers(3),
		wrapped: fmt.Errorf(s, a...),
	}
}

func (er *NError) Format(st fmt.State, verb rune) {
	switch verb {
	case 'v':
		if st.Flag('+') {
			ws := er.wrapped != nil || er.stack != nil

			if ws {
				_, _ = fmt.Fprintf(st, "%s", er.msg)
			}

			if er.stack != nil {
				er.stack.Format(st, verb)
			}

			if er.wrapped != nil {
				var d string
				if len(er.msg) > 0 {
					d = "; "
				}
				_, _ = fmt.Fprintf(st, "%s%+v", d, er.wrapped)
			}

			if ws {
				return
			}
		}

		fallthrough
	case 's':
		_, _ = io.WriteString(st, er.Error())
	case 'q':
		_, _ = fmt.Fprintf(st, "%q", er.Error())
	}
}

func (er *NError) Merge(err error) *NError {
	return &NError{
		id:      er.id,
		msg:     er.msg,
		wrapped: err,
	}
}

func (er *NError) Call() *NError {
	return &NError{
		id:    er.id,
		msg:   er.msg,
		stack: callers(3),
	}
}

func (er *NError) Caller(n int) *NError {
	return &NError{
		id:      er.id,
		msg:     er.msg,
		stack:   callers(n),
		wrapped: er.wrapped,
	}
}

func (er *NError) StackTrace() errors.StackTrace {
	if er.stack != nil {
		return er.stack.StackTrace()
	}

	if er.wrapped == nil {
		return nil
	}

	i, ok := er.wrapped.(stackTracer) // nolint:errorlint
	if !ok {
		return nil
	}

	return i.StackTrace()
}

// callers is from
// https://github.com/pkg/errors/blob/856c240a51a2bf8fb8269ea7f3f9b046aadde36e/stack.go#L163
func callers(skip int) *stack {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])
	var st stack = pcs[0:n]
	return &st
}

type stack []uintptr

func (s *stack) Format(st fmt.State, verb rune) {
	if verb == 'v' && st.Flag('+') {
		for _, pc := range *s {
			_, _ = fmt.Fprintf(st, "\n%+v", errors.Frame(pc))
		}
	}
}

func (s *stack) StackTrace() errors.StackTrace {
	f := make([]errors.Frame, len(*s))
	for i := 0; i < len(f); i++ {
		f[i] = errors.Frame((*s)[i])
	}
	return f
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}
