package errors

import "strings"

type MultiError interface {
	List() []error
	Size() int
	Add(error)
	Error() string
}

func NewMultiError() MultiError {
	return &multiError{
		errors: make([]error, 0),
	}
}

func MultiErrorWith(err error) MultiError {
	return &multiError{
		errors: []error{err},
	}
}

type multiError struct {
	errors []error
}

func (e *multiError) Size() int {
	return len(e.errors)
}

func (e *multiError) List() []error {
	return e.errors
}

func (e *multiError) Add(err error) {
	e.errors = append(e.errors, err)
}

func (e *multiError) Error() string {
	var builder strings.Builder
	for _, err := range e.errors {
		builder.WriteString(err.Error())
		builder.WriteRune('\n')
	}
	return builder.String()
}
