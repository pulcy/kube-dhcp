package main

import (
	"sync"
	"time"

	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
)

type memoryLeaseRegistry struct {
	mutex  sync.Mutex
	leases map[string]Lease
}

// NewMemoryLeaseRegistry creates an in-memory implementation of the LeaseRegistry.
func NewMemoryLeaseRegistry() LeaseRegistry {
	return &memoryLeaseRegistry{
		leases: make(map[string]Lease),
	}
}

// Get the lease for the given IP
func (r *memoryLeaseRegistry) GetByIP(ip string) (*Lease, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	l, found := r.leases[ip]
	if found {
		return &l, nil
	}
	return nil, maskAny(LeaseNotFoundError)
}

// Get all the leases for the given hardware address
func (r *memoryLeaseRegistry) ListByCHAddr(chAddr string) ([]Lease, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var result []Lease
	for _, l := range r.leases {
		if l.CHAddr == chAddr {
			result = append(result, l)
		}
	}
	return result, nil
}

// Remove the given lease
func (r *memoryLeaseRegistry) Remove(l *Lease) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.leases, l.IP)
	return nil
}

// Create a lease with given IP, hardware address and time to live.
func (r *memoryLeaseRegistry) Create(ip string, chAddr string, ttl time.Duration) (*Lease, error) {
	t := time.Now().Add(ttl)
	seconds := t.Unix()
	nanos := int32(t.UnixNano())
	l := Lease{
		IP:     ip,
		CHAddr: chAddr,
		ExpiratesAt: metav1.Time{
			Seconds: &seconds,
			Nanos:   &nanos,
		},
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.leases[ip] = l
	return &l, nil
}
