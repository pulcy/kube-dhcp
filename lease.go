package main

import (
	"context"
	"time"

	"github.com/ericchiang/k8s"
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
	Metadata    *metav1.ObjectMeta `json:"metadata"`
	IP          string             `json:"ip"`         // Leased IP address
	CHAddr      string             `json:"chaddr"`     // Client's hardware address
	ExpiratesAt metav1.Time        `json:"expires-at"` // When the lease expires
}

// GetMetadata is required to implement k8s.Resource
func (l *Lease) GetMetadata() *metav1.ObjectMeta {
	return l.Metadata
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

// LeaseList is a k8s list of Lease's
type LeaseList struct {
	Metadata *metav1.ListMeta `json:"metadata"`
	Items    []Lease          `json:"items"`
}

// GetMetadata is required to implement k8s.Resource
func (l *LeaseList) GetMetadata() *metav1.ListMeta {
	return l.Metadata
}

// LeaseRegistry abstracts a registry of leases.
type LeaseRegistry interface {
	// Get the lease for the given IP
	GetByIP(ctx context.Context, ip string) (*Lease, error)
	// Get all leases for the given hardware address
	ListByCHAddr(ctx context.Context, chAddr string) ([]Lease, error)
	// Remove the given lease
	Remove(ctx context.Context, l *Lease) error
	// Create a lease with given IP, hardware address and time to live.
	Create(ctx context.Context, ip string, chAddr string, ttl time.Duration) (*Lease, error)
}

func init() {
	// Register resources with the k8s package.
	k8s.Register("dhcp.pulcy.com", "v1", "leases", false, &Lease{})
	k8s.RegisterList("dhcp.pulcy.com", "v1", "leases", false, &LeaseList{})
}
