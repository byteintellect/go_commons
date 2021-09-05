package errors

import (
	"errors"
	"fmt"
)

type CFError struct {
	error
	Status uint32 `json:"status"`
	Code   string `json:"code"`
}

func NewCFError(options ...CFErrorOption) *CFError {
	e := &CFError{}
	for _, option := range options {
		option(e)
	}
	return e
}

func (cf *CFError) String() string {
	return fmt.Sprintf("{\"error\":\"%v\"," +
		                       "\"status\": \"%v\","+
		                       "\"code\": \"%v\""+
		                       "}", cf.error, cf.Status, cf.Code)
}

type CFErrorOption func(cfe *CFError)

func WithMessage(message string) CFErrorOption {
	return func(cfe *CFError) {
		cfe.error = errors.New(message)
	}
}

func WithStatus(status uint32) CFErrorOption {
	return func(cfe *CFError) {
		cfe.Status = status
	}
}

func WithCode(code string) CFErrorOption {
	return func(cfe *CFError) {
		cfe.Code = code
	}
}
