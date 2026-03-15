package apperr

import "fmt"

type DuplicateResourceError struct {
	Message string
}

func NewDuplicateResourceError(message string) *DuplicateResourceError {
	return &DuplicateResourceError{Message: message}
}

func (e *DuplicateResourceError) Error() string {
	return fmt.Sprintf("duplicate resource error: %v", e.Message)
}
