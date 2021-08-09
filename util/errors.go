package util

import (
	"fmt"
)

var (
	IgnoreError     = NewError("ignore")
	NotFoundError   = NewError("not found")
	FoundError      = NewError("found")
	DuplicatedError = NewError("duplicated error")
	WrongTypeError  = NewError("wrong type")
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
