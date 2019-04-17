package common

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Error struct {
	code    string `json:"code"`
	message string `json:"message"`
}

func (e Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"code":    e.code,
		"message": e.message,
	})
}

func (e *Error) UnmarshalJSON(b []byte) error {
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	var code, message string
	if c, ok := m["code"]; !ok {
		return errors.New("failed to Unmarshal Error; missing `code`")
	} else {
		code = c
	}

	if c, ok := m["message"]; !ok {
		return errors.New("failed to Unmarshal Error; missing `message`")
	} else {
		message = c
	}

	e.code = code
	e.message = message

	return nil
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
