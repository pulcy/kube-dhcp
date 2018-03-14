package main

import (
	"time"

	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
	"github.com/pkg/errors"
)

var (
	// LeaseNotFoundError is the error that is returned when a lease cannot be found.
	LeaseNotFoundError = errors.New("lease not found")
)

// IsLeaseNotFound returns true if the given error is or is caused by a LeaseNotFoundError.
func IsLeaseNotFound(err error) bool {
	return errors.Cause(err) == LeaseNotFoundError
}

// Lease is a single IP address claim
type Lease struct {
	IP          string      `json:"ip"`         // Leased IP address
	CHAddr      string      `json:"chaddr"`     // Client's hardware address
	ExpiratesAt metav1.Time `json:"expires-at"` // When the lease expires
}

// GetExpiresAt returns the expiration time of the lease
func (l Lease) GetExpiresAt() time.Time {
	seconds := l.ExpiratesAt.GetSeconds()
	nanos := int64(l.ExpiratesAt.GetNanos())
	return time.Unix(seconds, nanos)
}

// IsExpired returns true when the lease is expired,
// false otherwise.
func (l Lease) IsExpired() bool {
	return l.GetExpiresAt().Before(time.Now())
}

// LeaseRegistry abstracts a registry of leases.
type LeaseRegistry interface {
	// Get the lease for the given IP
	GetByIP(ip string) (*Lease, error)
	// Get all leases for the given hardware address
	ListByCHAddr(chAddr string) ([]Lease, error)
	// Remove the given lease
	Remove(l *Lease) error
	// Create a lease with given IP, hardware address and time to live.
	Create(ip string, chAddr string, ttl time.Duration) (*Lease, error)
}
