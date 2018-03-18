package util

import (
	"net/http"

	"github.com/ericchiang/k8s"
	"github.com/pkg/errors"
)

// IsK8sConflict returns true if the given error is or is caused by a kubernetes conflict error.
func IsK8sConflict(err error) bool {
	if apiErr, ok := errors.Cause(err).(*k8s.APIError); ok {
		return apiErr.Code == http.StatusConflict && apiErr.Status != nil && apiErr.Status.GetReason() == "Conflict"
	}
	return false
}

// IsK8sAlreadyExists returns true if the given error is or is caused by a kubernetes not-found error.
func IsK8sAlreadyExists(err error) bool {
	if apiErr, ok := errors.Cause(err).(*k8s.APIError); ok {
		return apiErr.Code == http.StatusConflict && apiErr.Status != nil && apiErr.Status.GetReason() == "AlreadyExists"
	}
	return false
}

// IsK8sNotFound returns true if the given error is or is caused by a kubernetes not-found error.
func IsK8sNotFound(err error) bool {
	if apiErr, ok := errors.Cause(err).(*k8s.APIError); ok {
		return apiErr.Code == http.StatusNotFound
	}
	return false
}
