package common

import (
	"encoding/json"
	"fmt"
)

type Error struct {
	code    string `json:"code"`
	message string `json:"message"`
}

func (e Error) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}

func (e Error) Code() string {
	return e.code
}

func (e Error) Message() string {
	return e.message
}

func (e Error) SetMessage(m string) Error {
	return Error{code: e.code, message: m}
}

func (e Error) Equal(n error) bool {
	ne, found := n.(Error)
	if found {
		return e.Code() == ne.Code()
	}

	return false
}

func NewError(name string, number uint, message string) Error {
	return Error{code: fmt.Sprintf("%s-%d", name, number), message: message}
}
