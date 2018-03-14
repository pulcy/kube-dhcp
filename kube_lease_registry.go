package main

import (
	"context"
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/ericchiang/k8s"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
)

type kubeLeaseRegistry struct {
	cli *k8s.Client
}

const (
	labelIP     = "dhcp.pulcy.com/ip"
	labelCHAddr = "dhcp.pulcy.com/chAddr"
)

// NewKubeLeaseRegistry creates an implementation of the LeaseRegistry
// that stores leases as Lease object in the kubernetes API server.
func NewKubeLeaseRegistry(cli *k8s.Client) LeaseRegistry {
	return &kubeLeaseRegistry{
		cli: cli,
	}
}

// Get the lease for the given IP
func (r *kubeLeaseRegistry) GetByIP(ctx context.Context, ip string) (*Lease, error) {
	sel := new(k8s.LabelSelector)
	sel.Eq(labelIP, ip)

	var leases LeaseList
	if err := r.cli.List(ctx, "", &leases, sel.Selector()); err != nil {
		return nil, maskAny(err)
	}
	if len(leases.Items) > 0 {
		return &leases.Items[0], nil
	}
	return nil, maskAny(LeaseNotFoundError)
}

// Get all the leases for the given hardware address
func (r *kubeLeaseRegistry) ListByCHAddr(ctx context.Context, chAddr string) ([]Lease, error) {
	sel := new(k8s.LabelSelector)
	sel.Eq(labelCHAddr, chAddr)

	var leases LeaseList
	if err := r.cli.List(ctx, "", &leases, sel.Selector()); err != nil {
		return nil, maskAny(err)
	}

	return leases.Items, nil
}

// Remove the given lease
func (r *kubeLeaseRegistry) Remove(ctx context.Context, l *Lease) error {
	if err := r.cli.Delete(ctx, l); err != nil {
		return maskAny(err)
	}
	return nil
}

// Create a lease with given IP, hardware address and time to live.
func (r *kubeLeaseRegistry) Create(ctx context.Context, ip string, chAddr string, ttl time.Duration) (*Lease, error) {
	t := time.Now().Add(ttl)
	seconds := t.Unix()
	nanos := int32(t.UnixNano())
	name := fmt.Sprintf("lease-%0x", sha1.Sum([]byte(fmt.Sprintf("%s-%s", ip, chAddr))))
	l := &Lease{
		Metadata: &metav1.ObjectMeta{
			Name: k8s.String(name),
			Labels: map[string]string{
				labelIP:     ip,
				labelCHAddr: chAddr,
			},
		},
		IP:     ip,
		CHAddr: chAddr,
		ExpiratesAt: metav1.Time{
			Seconds: &seconds,
			Nanos:   &nanos,
		},
	}

	if err := r.cli.Create(ctx, l); err != nil {
		return nil, maskAny(err)
	}
	return l, nil
}
