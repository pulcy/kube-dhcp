package main

import (
	"context"
	"log"
	"math/rand"
	"net"
	"time"

	dhcp "github.com/krolaw/dhcp4"
)

// NewHandler creates a DHCP handler for the given config
func NewHandler(config DHCPConfig, registry LeaseRegistry) (*DHCPHandler, error) {
	handler := &DHCPHandler{
		ip:             parseIP(config.ServerIP),
		leaseDuration:  2 * time.Hour,
		ranges:         config.Ranges,
		defaultOptions: config.Options,
		leases:         registry,
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

type DHCPHandler struct {
	ip             net.IP // Server IP to use
	defaultOptions DHCPOptions
	ranges         []AddressRange
	leaseDuration  time.Duration // Lease period
	leases         LeaseRegistry
}

// ServeDHCP serves DHCP requests.
func (h *DHCPHandler) ServeDHCP(p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) (d dhcp.Packet) {
	switch msgType {

	case dhcp.Discover:
		ip, nic := "", p.CHAddr().String()
		log.Printf("Discover: ip=%s nic=%s options=%v\n", ip, nic, options)
		// Find current leases
		ctx := context.Background()
		if list, err := h.leases.ListByCHAddr(ctx, nic); err == nil && len(list) > 0 {
			ip = list[0].IP
		}
		if ip == "" {
			ip = h.findFreeLease()
		}
		if ip != "" {
			ip4 := parseIP(ip)
			replyOpts := h.buildOptions(ip4)
			log.Printf("Discover: Offering ip=%s options=%v\n", ip, replyOpts)
			return dhcp.ReplyPacket(p, dhcp.Offer, h.ip, ip4, h.leaseDuration,
				replyOpts.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))
		}
		log.Println("Discover: No free IP found")

	case dhcp.Request:
		log.Printf("Request: nic=%s options=%v\n", p.CHAddr().String(), options)
		if server, ok := options[dhcp.OptionServerIdentifier]; ok && !net.IP(server).Equal(h.ip) {
			return nil // Message not for this dhcp server
		}
		reqIP := net.IP(options[dhcp.OptionRequestedIPAddress])
		if reqIP == nil {
			reqIP = net.IP(p.CIAddr())
		}

		if len(reqIP) == 4 && !reqIP.Equal(net.IPv4zero) {
			if h.isInRange(reqIP) {
				ctx := context.Background()
				ip := reqIP.String()
				chAddr := p.CHAddr().String()
				l, err := h.leases.GetByIP(ctx, ip)
				if IsLeaseNotFound(err) || ((err == nil) && l.CHAddr == chAddr) {
					_, err := h.leases.Create(ctx, ip, chAddr, h.leaseDuration)
					if err == nil {
						replyOpts := h.buildOptions(reqIP)
						return dhcp.ReplyPacket(p, dhcp.ACK, h.ip, reqIP, h.leaseDuration,
							replyOpts.SelectOrderOrAll(options[dhcp.OptionParameterRequestList]))
					}
					log.Printf("Failed to create lease for IP '%s': %v\n", ip, err)
				}
			}
		}
		return dhcp.ReplyPacket(p, dhcp.NAK, h.ip, nil, 0, nil)

	case dhcp.Release, dhcp.Decline:
		nic := p.CHAddr().String()
		ctx := context.Background()
		log.Printf("Release/Decline: nic=%s\n", nic)
		leases, err := h.leases.ListByCHAddr(ctx, nic)
		if err != nil {
			log.Printf("Failed to list leases for '%s': %v\n", nic, err)
		} else {
			for _, l := range leases {
				if err := h.leases.Remove(ctx, &l); err != nil {
					log.Printf("Failed to remove lease '%s': %v\n", l.IP, err)
				}
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
	ctx := context.Background()
	for _, rIdx := range rangePerms {
		r := h.ranges[rIdx]
		start := parseIP(r.Start)
		offsetPerm := rand.Perm(r.Length)
		for _, ofs := range offsetPerm {
			ip := dhcp.IPAdd(start, ofs).String()
			l, err := h.leases.GetByIP(ctx, ip)
			if IsLeaseNotFound(err) {
				return ip
			}
			if err == nil && l.IsExpired() {
				// Existing lease is expired
				err := h.leases.Remove(ctx, l)
				if err == nil {
					return l.IP
				}
				log.Printf("Failed to remove lease '%s': %v\n", ip, err)
			}
		}
	}
	return ""
}

// buildOptions creates a set of options for the given IP.
func (h *DHCPHandler) buildOptions(ip net.IP) dhcp.Options {
	options := make(dhcp.Options)
	config := h.defaultOptions
	subnetMask := config.SubnetMask
	if subnetMask == "" {
		subnetMask = "255.255.255.0"
	}
	options[dhcp.OptionSubnetMask] = parseIP(subnetMask)
	if config.RouterIP != "" {
		options[dhcp.OptionRouter] = parseIP(config.RouterIP)
	}
	if config.DNSServerIP != "" {
		options[dhcp.OptionDomainNameServer] = parseIP(config.DNSServerIP)
	}
	if config.DomainName != "" {
		options[dhcp.OptionDomainName] = []byte(config.DomainName)
	}
	return options
}
