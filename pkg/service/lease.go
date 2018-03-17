package service

import (
	"context"
	"time"
)

// Lease is a single IP address claim
type Lease interface {
	GetIP() string
	GetCHAddr() string
	GetHostName() string
	IsExpired() bool
}

// LeaseRegistry abstracts a registry of leases.
type LeaseRegistry interface {
	// Get the lease for the given IP
	GetByIP(ctx context.Context, ip string) (Lease, error)
	// Get all leases for the given hardware address
	ListByCHAddr(ctx context.Context, chAddr string) ([]Lease, error)
	// Remove the given lease
	Remove(ctx context.Context, l Lease) error
	// Create a lease with given IP, hardware address and time to live.
	Create(ctx context.Context, ip, chAddr, hostname string, ttl time.Duration) (Lease, error)
}
