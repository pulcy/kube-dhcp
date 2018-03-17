package service

import (
	"github.com/pkg/errors"
)

var (
	// LeaseNotFoundError is the error that is returned when a lease cannot be found.
	LeaseNotFoundError = errors.New("lease not found")

	maskAny = errors.WithStack
)

// IsLeaseNotFound returns true if the given error is or is caused by a LeaseNotFoundError.
func IsLeaseNotFound(err error) bool {
	return errors.Cause(err) == LeaseNotFoundError
}
