package netutil

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/vehkiya/qbit-gluetun-sync/pkg/logger"
)

// ParseAllowedIPs parses a comma-separated list of CIDR strings into a slice of *net.IPNet.
// If an IP is provided without a mask, it defaults to /32 for IPv4 and /128 for IPv6.
func ParseAllowedIPs(ips string) ([]*net.IPNet, error) {
	if ips == "" {
		return nil, nil
	}
	var nets []*net.IPNet
	for _, ipStr := range strings.Split(ips, ",") {
		ipStr = strings.TrimSpace(ipStr)
		if ipStr == "" {
			continue
		}

		// Remove brackets if someone provided an IPv6 address in brackets (e.g. [::1] or [::1]/128)
		ipStr = strings.ReplaceAll(ipStr, "[", "")
		ipStr = strings.ReplaceAll(ipStr, "]", "")

		// If it's a single IP without a mask, add /32 for IPv4 or /128 for IPv6
		if !strings.Contains(ipStr, "/") {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP address: %s", ipStr)
			}
			if ip.To4() != nil {
				ipStr += "/32"
			} else {
				ipStr += "/128"
			}
		}
		_, ipNet, err := net.ParseCIDR(ipStr)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR: %s: %w", ipStr, err)
		}
		nets = append(nets, ipNet)
	}
	return nets, nil
}

// IsAllowedIP checks if the given remote IP address string is within any of the allowed IP networks.
func IsAllowedIP(allowedIPs []*net.IPNet, remoteAddr string) bool {
	// If allowedIPs is nil or empty, access is denied by default (fail-closed)
	if len(allowedIPs) == 0 {
		return false
	}

	ipStr, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// If SplitHostPort fails, it's likely because the port is missing.
		// The address might be a bare IP or a bracketed IPv6, e.g., `[::1]`.
		ipStr = remoteAddr
		// net.ParseIP doesn't handle brackets, so we need to trim them.
		if len(ipStr) > 1 && ipStr[0] == '[' && ipStr[len(ipStr)-1] == ']' {
			ipStr = ipStr[1 : len(ipStr)-1]
		}
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		logger.Warn("Failed to parse remote IP address", "remoteAddr", remoteAddr)
		return false
	}

	for _, network := range allowedIPs {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// IPAllowlistMiddleware wraps an http.Handler to enforce IP restrictions.
func IPAllowlistMiddleware(allowedIPs []*net.IPNet, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAllowedIP(allowedIPs, r.RemoteAddr) {
			logger.Warn("Blocked unauthorized request",
				"ip", r.RemoteAddr,
				"path", r.URL.Path,
				"method", r.Method,
			)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
