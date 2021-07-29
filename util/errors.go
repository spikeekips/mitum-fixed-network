package util

import (
	"fmt"

	"github.com/spikeekips/mitum/util/errors"
)

// NOTE Generaal Errors

var IgnoreError = errors.NewError("ignore")

// Data Errors

var (
	NotFoundError   = errors.NewError("not found")
	FoundError      = errors.NewError("found")
	DuplicatedError = errors.NewError("duplicated error")
	WrongTypeError  = errors.NewError("wrong type")
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
