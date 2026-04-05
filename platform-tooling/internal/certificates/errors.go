package certificates

import "errors"

type ErrorClass string

const (
	ErrorClassSchema   ErrorClass = "schema_error"
	ErrorClassParse    ErrorClass = "parse_error"
	ErrorClassKernel   ErrorClass = "kernel_validation_error"
	ErrorClassInternal ErrorClass = "internal_error"
)

type ClassifiedError struct {
	Class ErrorClass
	Err   error
}

func (e *ClassifiedError) Error() string {
	if e.Err == nil {
		return string(e.Class)
	}
	return e.Err.Error()
}

func (e *ClassifiedError) Unwrap() error {
	return e.Err
}

func NewError(class ErrorClass, err error) error {
	if err == nil {
		return nil
	}
	return &ClassifiedError{
		Class: class,
		Err:   err,
	}
}

func ClassOf(err error) ErrorClass {
	if err == nil {
		return ""
	}

	if classified, ok := errors.AsType[*ClassifiedError](err); ok {
		return classified.Class
	}

	return ErrorClassInternal
}
