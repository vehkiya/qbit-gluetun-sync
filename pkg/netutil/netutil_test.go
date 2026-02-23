package netutil

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseAllowedIPs(t *testing.T) {
	tests := []struct {
		name     string
		ips      string
		expected int
		wantErr  bool
	}{
		{"Empty string", "", 0, false},
		{"Empty value (spaces)", "   ", 0, false},
		{"Invalid value", "not-an-ip", 0, true},
		{"Single IP IPv4", "192.168.1.1", 1, false},
		{"Single IP IPv6", "2001:db8::1", 1, false},
		{"Single IP IPv6 (bracketed)", "[2001:db8::1]", 1, false},
		{"Single CIDR IPv4", "192.168.1.0/24", 1, false},
		{"Single CIDR IPv6", "2001:db8::/32", 1, false},
		{"Multiple distinct", "10.0.0.0/8, 172.16.0.0/12,192.168.0.0/16", 3, false},
		{"Combo of IPv4, IPv6 and IPv6 bracketed", "192.168.1.1, 2001:db8::1, [fe80::1]", 3, false},
		{"Trailing comma", "192.168.1.1,", 1, false},
		{"With invalid", "10.0.0.0/8, invalid-ip", 0, true},
		{"Invalid CIDR", "192.168.1.0/33", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseAllowedIPs(tt.ips)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAllowedIPs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(result) != tt.expected {
				t.Errorf("Expected %d networks, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestIsAllowedIP(t *testing.T) {
	allowedIPStr := "192.168.1.0/24, 10.0.0.1, 2001:db8::/32"
	allowedIPs, _ := ParseAllowedIPs(allowedIPStr)

	tests := []struct {
		name       string
		allowlist  []*net.IPNet
		remoteAddr string
		expected   bool
	}{
		{"Empty allowlist denies all", nil, "8.8.8.8:1234", false},
		{"Empty slice denies all", []*net.IPNet{}, "8.8.8.8:1234", false},
		{"Allowed IPv4 subnet", allowedIPs, "192.168.1.50:5678", true},
		{"Allowed specific IPv4", allowedIPs, "10.0.0.1:9090", true},
		{"Allowed IPv6 subnet", allowedIPs, "[2001:db8::1]:1234", true},
		{"Denied IPv4 (different subnet)", allowedIPs, "192.168.2.1:1234", false},
		{"Denied IPv4 (public)", allowedIPs, "8.8.8.8:1234", false},
		{"Denied IPv6", allowedIPs, "[2002:db8::1]:1234", false},
		{"Invalid remote IP", allowedIPs, "invalid-ip", false},
		{"Valid IPv4 without port", allowedIPs, "192.168.1.50", true},
		{"Valid IPv6 without port", allowedIPs, "2001:db8::1", true},
		{"Valid bracketed IPv6 without port", allowedIPs, "[2001:db8::1]", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAllowedIP(tt.allowlist, tt.remoteAddr)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.remoteAddr, result)
			}
		})
	}
}

func TestIPAllowlistMiddleware(t *testing.T) {
	allowedIPs, _ := ParseAllowedIPs("192.168.1.0/24")

	handlerFunc := IPAllowlistMiddleware(allowedIPs, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	// Test allowed request
	reqAllowed := httptest.NewRequest("GET", "/test", nil)
	reqAllowed.RemoteAddr = "192.168.1.100:1234"
	wAllowed := httptest.NewRecorder()
	handlerFunc.ServeHTTP(wAllowed, reqAllowed)
	if wAllowed.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK for allowed IP, got %d", wAllowed.Code)
	}

	// Test denied request
	reqDenied := httptest.NewRequest("GET", "/test", nil)
	reqDenied.RemoteAddr = "8.8.8.8:5678"
	wDenied := httptest.NewRecorder()
	handlerFunc.ServeHTTP(wDenied, reqDenied)
	if wDenied.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 Forbidden for denied IP, got %d", wDenied.Code)
	}
}
