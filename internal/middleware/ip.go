package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
)

var httpInterestingHeaders = []string{
	"CF-Connecting-IP",
	"X-Forwarded-For",
	"X-Real-IP",
}

func getProxyClientIP(r *http.Request) string {
	var originIP string

	for _, interestingHeader := range httpInterestingHeaders {
		originIP = r.Header.Get(interestingHeader)

		// check if the header exists
		if originIP = strings.TrimSpace(originIP); originIP == "" {
			continue
		}

		// if it contains multiple values - if it's "X-Forwarded-For") and return the first address in the comma-separated list of IPs
		originIP, _, _ = strings.Cut(originIP, ",")
		originIP = strings.TrimSpace(originIP)

		parsedIP := net.ParseIP(originIP)
		// isPrivateIP prevents spoofing attempts
		if parsedIP == nil || isPrivateIP(parsedIP) {
			// interesting header exists but IP is nil or private
			continue
		}

		// ip is good, comes through proxy
		return originIP
	}

	// ip does not have proxy headers
	return getDirectClientIPValidated(r)
}

func getDirectClientIPValidated(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// r.RemoteAddr does not have a port, return as is
		ip = r.RemoteAddr
	}

	ip = strings.TrimSpace(ip)

	if net.ParseIP(ip) == nil {
		return "" // Invalid IP - let middleware handle this
	}
	return ip
}

var getPrivateIPBlocks = sync.OnceValue(func() []*net.IPNet {
	privateCIDRnets := []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local
		"fe80::/10",      // IPv6 link-local
	}

	blocks := make([]*net.IPNet, 0, len(privateCIDRnets))
	for _, cidr := range privateCIDRnets {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		blocks = append(blocks, ipNet)
	}

	return blocks
})

func isPrivateIP(ip net.IP) bool {
	for _, block := range getPrivateIPBlocks() {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}
