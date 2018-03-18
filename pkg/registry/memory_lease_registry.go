package registry

import (
	"context"
	"sync"
	"time"

	"github.com/pulcy/kube-dhcp/pkg/service"
)

type memoryLeaseRegistry struct {
	mutex  sync.Mutex
	leases map[string]memoryLease
}

type memoryLease struct {
	IP          string
	CHAddr      string
	HostName    string
	ExpiratesAt time.Time
}

func (l memoryLease) GetIP() string {
	return l.IP
}

func (l memoryLease) GetCHAddr() string {
	return l.CHAddr
}

func (l memoryLease) GetHostName() string {
	return l.HostName
}

func (l memoryLease) IsExpired() bool {
	return l.ExpiratesAt.Before(time.Now())
}

// NewMemoryLeaseRegistry creates an in-memory implementation of the LeaseRegistry.
func NewMemoryLeaseRegistry() service.LeaseRegistry {
	return &memoryLeaseRegistry{
		leases: make(map[string]memoryLease),
	}
}

// Get the lease for the given IP
func (r *memoryLeaseRegistry) GetByIP(ctx context.Context, ip string) (service.Lease, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	l, found := r.leases[ip]
	if found {
		return &l, nil
	}
	return nil, maskAny(service.LeaseNotFoundError)
}

// Get all the leases for the given hardware address
func (r *memoryLeaseRegistry) ListByCHAddr(ctx context.Context, chAddr string) ([]service.Lease, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var result []service.Lease
	for _, l := range r.leases {
		if l.CHAddr == chAddr {
			result = append(result, l)
		}
	}
	return result, nil
}

// Remove the given lease
func (r *memoryLeaseRegistry) Remove(ctx context.Context, l service.Lease) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.leases, l.GetIP())
	return nil
}

// Create a lease with given IP, hardware address and time to live.
func (r *memoryLeaseRegistry) Create(ctx context.Context, ip, chAddr, hostname string, ttl time.Duration) (service.Lease, error) {
	t := time.Now().Add(ttl)
	l := memoryLease{
		IP:          ip,
		CHAddr:      chAddr,
		ExpiratesAt: t,
		HostName:    hostname,
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.leases[ip] = l
	return &l, nil
}
