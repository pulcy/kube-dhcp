package main

import (
	"context"
	"math/rand"
	"net"
	"time"

	dhcp "github.com/krolaw/dhcp4"
)

// NewHandler creates a DHCP handler for the given config
func NewHandler(config DHCPConfig) (*DHCPHandler, error) {
	handler := &DHCPHandler{
		ip:            net.ParseIP(config.ServerIP),
		leaseDuration: 2 * time.Hour,
		ranges:        config.Ranges,
		leases:        make(map[string]lease, 10),
		options:       dhcp.Options{},
	}
	if config.Options.SubnetMask != "" {
		handler.options[dhcp.OptionSubnetMask] = net.ParseIP(config.Options.SubnetMask)
	} else {
		handler.options[dhcp.OptionSubnetMask] = net.ParseIP("255.255.255.0")
	}
	if config.Options.RouterIP != "" {
		handler.options[dhcp.OptionRouter] = net.ParseIP(config.Options.RouterIP)
	}
	if config.Options.DNSServerIP != "" {
		handler.options[dhcp.OptionDomainNameServer] = net.ParseIP(config.Options.DNSServerIP)
	}
	return handler, nil
}

// Run the handler until the given context is canceled.
func (h *DHCPHandler) Run(ctx context.Context) error {
	l, err := net.ListenPacket("udp4", ":67")
	if err != nil {
		return maskAny(err)
	}
	defer l.Close()

	errors := make(chan error, 1)
	go func() {
		defer close(errors)
		if err := dhcp.Serve(l, h); err != nil {
			errors <- err
		}
	}()

	select {
	case err := <-errors:
		return maskAny(err)
	case <-ctx.Done():
		// Context closed
		return nil
	}
}

type lease struct {
	nic    string    // Client's CHAddr
	expiry time.Time // When the lease expires
}

type DHCPHandler struct {
	ip            net.IP       // Server IP to use
	options       dhcp.Options // Options to send to DHCP Clients
	ranges        []AddressRange
	leaseDuration time.Duration    // Lease period
	leases        map[string]lease // Map to keep track of leases (ip->lease)
}

// ServeDHCP serves DHCP requests.
func (h *DHCPHandler) ServeDHCP(p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) (d dhcp.Packet) {
	switch msgType {

	case dhcp.Discover:
		ip, nic := "", p.CHAddr().String()
		for k, v := range h.leases { // Find previous lease
			if v.nic == nic {
				ip = k
				break
			}
		}
		if ip == "" {
			ip = h.findFreeLease()
		}
		if ip != "" {
			return dhcp.ReplyPacket(p, dhcp.Offer, h.ip, net.ParseIP(ip), h.leaseDuration,
				h.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))
		}

	case dhcp.Request:
		if server, ok := options[dhcp.OptionServerIdentifier]; ok && !net.IP(server).Equal(h.ip) {
			return nil // Message not for this dhcp server
		}
		reqIP := net.IP(options[dhcp.OptionRequestedIPAddress])
		if reqIP == nil {
			reqIP = net.IP(p.CIAddr())
		}

		if len(reqIP) == 4 && !reqIP.Equal(net.IPv4zero) {
			if h.isInRange(reqIP) {
				ip := reqIP.String()
				l, found := h.leases[ip]
				if !found || l.nic == p.CHAddr().String() {
					h.leases[ip] = lease{nic: p.CHAddr().String(), expiry: time.Now().Add(h.leaseDuration)}
					return dhcp.ReplyPacket(p, dhcp.ACK, h.ip, reqIP, h.leaseDuration,
						h.options.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))
				}
			}
		}
		return dhcp.ReplyPacket(p, dhcp.NAK, h.ip, nil, 0, nil)

	case dhcp.Release, dhcp.Decline:
		nic := p.CHAddr().String()
		for k, v := range h.leases {
			if v.nic == nic {
				delete(h.leases, k)
				break
			}
		}
	}
	return nil
}

// isInRange returns true when the given IP fits in one of the given address ranges.
func (h *DHCPHandler) isInRange(ip net.IP) bool {
	for _, r := range h.ranges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}

// findFreeLease tries to find a free IP address.
// Returns an empty string if no free address is found.
func (h *DHCPHandler) findFreeLease() string {
	rangePerms := rand.Perm(len(h.ranges))
	now := time.Now()
	for _, rIdx := range rangePerms {
		r := h.ranges[rIdx]
		start := net.ParseIP(r.Start)
		offsetPerm := rand.Perm(r.Length)
		for _, ofs := range offsetPerm {
			ip := dhcp.IPAdd(start, ofs).String()
			l, found := h.leases[ip]
			if !found {
				return ip
			}
			if l.expiry.Before(now) {
				// Existing lease is expired
				delete(h.leases, ip)
				return ip
			}
		}
	}
	return ""
}
