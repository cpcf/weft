package engine

import (
	"fmt"
	"strings"
)

type GenerationError struct {
	Path    string
	Message string
	Err     error
}

func (e *GenerationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Path, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

func (e *GenerationError) Unwrap() error {
	return e.Err
}

type MultiError struct {
	Errors []*GenerationError
}

func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return "no errors"
	}
	if len(m.Errors) == 1 {
		return m.Errors[0].Error()
	}
	
	var msgs []string
	for _, err := range m.Errors {
		msgs = append(msgs, err.Error())
	}
	return fmt.Sprintf("multiple errors:\n%s", strings.Join(msgs, "\n"))
}

func (m *MultiError) Add(path, message string, err error) {
	m.Errors = append(m.Errors, &GenerationError{
		Path:    path,
		Message: message,
		Err:     err,
	})
}

func (m *MultiError) HasErrors() bool {
	return len(m.Errors) > 0
}