package registry

import (
	"context"
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/ericchiang/k8s"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"

	"github.com/pulcy/kube-dhcp/pkg/service"
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
func NewKubeLeaseRegistry(cli *k8s.Client) service.LeaseRegistry {
	return &kubeLeaseRegistry{
		cli: cli,
	}
}

// Get the lease for the given IP
func (r *kubeLeaseRegistry) GetByIP(ctx context.Context, ip string) (service.Lease, error) {
	sel := new(k8s.LabelSelector)
	sel.Eq(labelIP, ip)

	var leases LeaseList
	if err := r.cli.List(ctx, k8s.AllNamespaces, &leases, sel.Selector()); err != nil {
		return nil, maskAny(err)
	}
	if len(leases.Items) > 0 {
		return leases.Items[0], nil
	}
	return nil, maskAny(service.LeaseNotFoundError)
}

// Get all the leases for the given hardware address
func (r *kubeLeaseRegistry) ListByCHAddr(ctx context.Context, chAddr string) ([]service.Lease, error) {
	sel := new(k8s.LabelSelector)
	sel.Eq(labelCHAddr, chAddr)

	var leases LeaseList
	if err := r.cli.List(ctx, k8s.AllNamespaces, &leases, sel.Selector()); err != nil {
		return nil, maskAny(err)
	}

	result := make([]service.Lease, 0, len(leases.Items))
	for _, l := range leases.Items {
		result = append(result, l)
	}
	return result, nil
}

// Remove the given lease
func (r *kubeLeaseRegistry) Remove(ctx context.Context, l service.Lease) error {
	lease := l.(Lease)
	if err := r.cli.Delete(ctx, &lease); err != nil {
		return maskAny(err)
	}
	return nil
}

// Create a lease with given IP, hardware address and time to live.
func (r *kubeLeaseRegistry) Create(ctx context.Context, ip, chAddr, hostname string, ttl time.Duration) (service.Lease, error) {
	t := time.Now().Add(ttl)
	seconds := t.Unix()
	//nanos := int32(t.UnixNano())
	name := fmt.Sprintf("lease-%0x", sha1.Sum([]byte(fmt.Sprintf("%s-%s", ip, chAddr))))
	l := Lease{
		Kind:       leaseKind,
		ApiVersion: apiGroup + "/" + apiVersion,
		Metadata: &metav1.ObjectMeta{
			Name:      k8s.String(name),
			Namespace: k8s.String(""),
			Labels: map[string]string{
				labelIP:     ip,
				labelCHAddr: chAddr,
			},
		},
		IP:          ip,
		CHAddr:      chAddr,
		ExpiratesAt: seconds,
		/* metav1.Time{
			Seconds: &seconds,
			Nanos:   &nanos,
		},*/
		HostName: hostname,
	}

	if err := r.cli.Create(ctx, &l); err != nil {
		return nil, maskAny(err)
	}
	return l, nil
}
