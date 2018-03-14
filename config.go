package main

import (
	"fmt"
	"net"

	dhcp "github.com/krolaw/dhcp4"
)

// DHCPConfig holds the configuration structure of the DHCP server.
type DHCPConfig struct {
	// ServerIP is the IP address of the server itself
	ServerIP string         `json:"server-ip"`
	Ranges   []AddressRange `json:"ranges"`
	Options  DHCPOptions    `json:"options"`
}

// DHCPOptions holds various options of the DHCP protocol
type DHCPOptions struct {
	SubnetMask  string `json:"subnet-mask,omitempty"`
	RouterIP    string `json:"router-ip,omitempty"`
	DNSServerIP string `json:"dns-ip,omitempty"`
	DomainName  string `json:"domain,omitempty"`
}

// Validate changes the values in the given config.
// Returns nil if all ok, otherwise an error.
func (o DHCPOptions) Validate() error {
	if o.SubnetMask != "" {
		if ip := parseIP(o.SubnetMask); ip == nil {
			return maskAny(fmt.Errorf("Failed to parse subnet-mask option '%s'", o.SubnetMask))
		}
	}
	if o.RouterIP != "" {
		if ip := parseIP(o.RouterIP); ip == nil {
			return maskAny(fmt.Errorf("Failed to parse router-ip option '%s'", o.RouterIP))
		}
	}
	if o.DNSServerIP != "" {
		if ip := parseIP(o.DNSServerIP); ip == nil {
			return maskAny(fmt.Errorf("Failed to parse dns-ip option '%s'", o.DNSServerIP))
		}
	}
	return nil
}

// AddressRange is a range of IP addresses that can be assigned.
type AddressRange struct {
	Start  string `json:"start"`  // First IP address
	Length int    `json:"length"` // Number of addresses in this range
}

// Validate changes the values in the given range.
// Returns nil if all ok, otherwise an error.
func (r AddressRange) Validate() error {
	ip := parseIP(r.Start)
	if ip == nil {
		return maskAny(fmt.Errorf("Failed to parse range start '%s'", r.Start))
	}
	if r.Length < 1 {
		return maskAny(fmt.Errorf("Range length must be >= 1, got %d", r.Length))
	}
	if int(ip[len(ip)-1])+r.Length > 255 {
		return maskAny(fmt.Errorf("Range length out of range, got %d", r.Length))
	}
	return nil
}

// Contains returns true when the given IP is part of this range, false otherwise.
func (r AddressRange) Contains(ip net.IP) bool {
	start := parseIP(r.Start)
	stop := dhcp.IPAdd(start, r.Length)
	return dhcp.IPInRange(start, stop, ip)
}

// Validate changes the values in the given config.
// Returns nil if all ok, otherwise an error.
func (c *DHCPConfig) Validate(defaultServerIP string) error {
	if c.ServerIP == "" {
		c.ServerIP = defaultServerIP
	}
	if ip := parseIP(c.ServerIP); ip == nil {
		return maskAny(fmt.Errorf("Failed to parse server-ip '%s'", c.ServerIP))
	}
	for _, r := range c.Ranges {
		if err := r.Validate(); err != nil {
			return maskAny(err)
		}
	}
	if err := c.Options.Validate(); err != nil {
		return maskAny(err)
	}
	return nil
}
