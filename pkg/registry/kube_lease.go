package registry

import (
	"time"

	"github.com/ericchiang/k8s"
	metav1 "github.com/ericchiang/k8s/apis/meta/v1"
)

const (
	leaseKind   = "Lease"
	leasePlural = "leases"
	apiGroup    = "dhcp.pulcy.com"
	apiVersion  = "v1"
)

// Lease is a single IP address claim
type Lease struct {
	Kind        string             `json:"kind,omitempty"`
	ApiVersion  string             `json:"apiVersion,omitempty"`
	Metadata    *metav1.ObjectMeta `json:"metadata"`
	IP          string             `json:"ip"`         // Leased IP address
	CHAddr      string             `json:"chaddr"`     // Client's hardware address
	HostName    string             `json:"hostname"`   // Hostname of the user of the lease
	ExpiratesAt int64              `json:"expires-at"` // When the lease expires
}

// GetMetadata is required to implement k8s.Resource
func (l *Lease) GetMetadata() *metav1.ObjectMeta {
	return l.Metadata
}

func (l Lease) GetIP() string {
	return l.IP
}

func (l Lease) GetCHAddr() string {
	return l.CHAddr
}

func (l Lease) GetHostName() string {
	return l.HostName
}

// GetExpiresAt returns the expiration time of the lease
func (l Lease) GetExpiresAt() time.Time {
	return time.Unix(l.ExpiratesAt, 0)
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

func init() {
	// Register resources with the k8s package.
	k8s.Register(apiGroup, apiVersion, leasePlural, false, &Lease{})
	k8s.RegisterList(apiGroup, apiVersion, leasePlural, false, &LeaseList{})
}
