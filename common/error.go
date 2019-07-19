package common

import (
	"encoding/json"
	"fmt"

	"golang.org/x/xerrors"
)

type ErrorCode uint

type Error struct {
	id      string
	message string
	err     error
	frame   xerrors.Frame
}

func NewError(errorType string, errorID ErrorCode, message string) Error {
	return Error{
		id:      fmt.Sprintf("%s-%d", errorType, errorID),
		message: message,
	}
}

func (e Error) Unwrap() error {
	return e.err
}

func (e Error) Format(s fmt.State, v rune) {
	xerrors.FormatError(e, s, v)
}

func (e Error) FormatError(p xerrors.Printer) error {
	p.Print(e.Error())
	e.frame.Format(p)
	return nil
}

func (e Error) Error() string {
	if e.err == nil {
		return e.message
	}

	return fmt.Sprintf("%s; %s", e.message, e.err.Error())
}

func (e Error) Is(err error) bool {
	var ae Error
	if !xerrors.As(err, &ae) {
		return false
	}

	return e.id == ae.id
}

func (e Error) New(err error) Error {
	return Error{
		id:      e.id,
		message: e.message,
		err:     err,
		frame:   xerrors.Caller(1),
	}
}

func (e Error) Newf(s string, args ...interface{}) Error {
	return Error{
		id:      e.id,
		message: e.message,
		err:     fmt.Errorf(s, args...),
		frame:   xerrors.Caller(1),
	}
}

func (e Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"id":        e.id,
		"message":   e.message,
		"traceback": fmt.Sprintf("%+v", e),
	})
}
