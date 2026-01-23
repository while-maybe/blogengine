package middleware

import (
	"net"
	"net/http"
	"strings"
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

		return strings.TrimSpace(originIP)
	}

	// Fallback to RemoteAddr
	return getDirectClientIPValidated(r)
}

func getDirectClientIPValidated(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// r.RemoteAddr does not have a port, return as is
		return r.RemoteAddr
	}

	if net.ParseIP(ip) == nil {
		return "" // Invalid IP - let middleware handle this
	}
	return ip
}
