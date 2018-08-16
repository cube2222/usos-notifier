package customerrors

import (
	"fmt"

	"github.com/pkg/errors"
)

type Permanent struct {
	Cause error
}

func NewPermanent(err error) error {
	return &Permanent{
		Cause: err,
	}
}

func (err *Permanent) Error() string {
	return fmt.Sprintf("permanent error: %v", err.Cause)
}

func IsPermanent(err error) bool {
	if _, ok := errors.Cause(err).(*Permanent); ok {
		return true
	} else {
		return false
	}
}
