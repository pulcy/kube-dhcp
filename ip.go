package main

import (
	"net"
)

// parseIP an IP address and reduce to 4 bytes for IPv4
// or 16 bytes if IPv6.
func parseIP(input string) net.IP {
	ip := net.ParseIP(input)
	if ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4
		}
		if ip6 := ip.To16(); ip6 != nil {
			return ip6
		}
	}
	return ip
}
