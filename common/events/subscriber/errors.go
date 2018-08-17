package subscriber

import (
	"fmt"

	"github.com/pkg/errors"
)

type NonRetryableError struct {
	cause error
}

func NewNonRetryableError(err error) error {
	return &NonRetryableError{
		cause: err,
	}
}

func (err *NonRetryableError) Error() string {
	return fmt.Sprintf("non retryable error: %v", err.cause)
}

func IsNonRetryableError(err error) bool {
	_, ok := errors.Cause(err).(*NonRetryableError)
	return ok
}
