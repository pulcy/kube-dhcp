package registry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ericchiang/k8s"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"

	"github.com/pulcy/kube-dhcp/pkg/service"
	"github.com/pulcy/kube-dhcp/pkg/util"
)

type kubeLeaseRegistry struct {
	cli *k8s.Client
}

var (
	ip2NameReplacer     = strings.NewReplacer(".", "-", ":", "-")
	chAddr2NameReplacer = strings.NewReplacer(":", "")
)

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
	sel.Eq(labelIP, ip2Name(ip))

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
	sel.Eq(labelCHAddr, chAddr2Name(chAddr))

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
	if err := r.cli.Delete(ctx, &lease); err != nil && !util.IsK8sNotFound(err) {
		return maskAny(err)
	}
	return nil
}

// Create a lease with given IP, hardware address and time to live.
func (r *kubeLeaseRegistry) Create(ctx context.Context, ip, chAddr, hostname string, ttl time.Duration) (service.Lease, error) {
	t := time.Now().Add(ttl)
	seconds := t.Unix()
	nanos := int32(t.UnixNano())
	name := fmt.Sprintf("lease-%s", ip2Name(ip))
	l := Lease{
		Kind:       leaseKind,
		APIVersion: apiGroup + "/" + apiVersion,
		Metadata: &metav1.ObjectMeta{
			Name:      k8s.String(name),
			Namespace: k8s.String(k8s.AllNamespaces),
			Labels: map[string]string{
				labelIP:     ip2Name(ip),
				labelCHAddr: chAddr2Name(chAddr),
			},
		},
		Spec: LeaseSpec{
			IP:     ip,
			CHAddr: chAddr,
			ExpiratesAt: metav1.Time{
				Seconds: &seconds,
				Nanos:   &nanos,
			},
			HostName: hostname,
		},
	}

	err := r.cli.Create(ctx, &l)
	if err == nil {
		// OK
		return l, nil
	}
	if util.IsK8sAlreadyExists(err) {
		// Lease resource exists, we must update it
		var current Lease
		if err = r.cli.Get(ctx, l.Metadata.GetNamespace(), l.Metadata.GetName(), &current); err == nil {
			// Now update
			current.Spec = l.Spec
			md := current.GetMetadata()
			if md.Labels == nil {
				md.Labels = make(map[string]string)
			}
			for k, v := range l.Metadata.Labels {
				md.Labels[k] = v
			}
			err := r.cli.Update(ctx, &current)
			if err == nil {
				return &current, nil
			}
			return nil, maskAny(err)
		}
	}
	return nil, maskAny(err)
}

// ip2Name converts the given IP address to a valid k8s name.
func ip2Name(ip string) string {
	return ip2NameReplacer.Replace(ip)
}

// chAddr2Name converts the given hardware address to a valid k8s name.
func chAddr2Name(chAddr string) string {
	return chAddr2NameReplacer.Replace(chAddr)
}
