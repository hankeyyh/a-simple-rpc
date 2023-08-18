package error

import (
	"fmt"
	"sync"
)

type MultiError struct {
	Errors []error
	mu     sync.Mutex
}

func NewMultiError(errors []error) *MultiError {
	return &MultiError{Errors: errors}
}

func (e *MultiError) Append(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Errors = append(e.Errors, err)
}

func (e *MultiError) Error() string {
	return fmt.Sprintf("%v", e.Errors)
}

func (e *MultiError) ErrorOrNil() error {
	if e == nil || len(e.Errors) == 0 {
		return nil
	}
	return e
}
