package middleware

import (
	"net/http"
	"testing"
)

func TestGetProxyClientIP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{
			name:     "direct connection (no headers)",
			headers:  map[string]string{},
			remote:   "192.168.0.1:999",
			expected: "192.168.0.1",
		},
		{
			name:     "direct connection whitespaces (no headers)",
			headers:  map[string]string{},
			remote:   "   192.168.0.1     :   999   ",
			expected: "192.168.0.1",
		},
		{
			name:     "proxy connection (header matches)",
			headers:  map[string]string{"CF-Connecting-IP": "20.55.20.55"},
			remote:   "192.168.0.1:999",
			expected: "20.55.20.55",
		},
		{
			name: "proxy connection (header precedence, multiple headers, multiple IPs)",
			headers: map[string]string{
				"CF-Connecting-IP": "20.55.20.55",
				"X-Forwarded-For":  "1.2.3.4, 5.6.7.8",
			},
			remote:   "192.168.0.1:999",
			expected: "20.55.20.55",
		},
		{
			name: "proxy connection whitespaces everywhere (header precedence, multiple headers, multiple IPs)",
			headers: map[string]string{
				"CF-Connecting-IP": "     20.55.20.55 ",
				"X-Forwarded-For":  "1.2.3.4     ,    5.6.7.8  ",
			},
			remote:   "192.168.0.1:999",
			expected: "20.55.20.55",
		},
		{
			name: "proxy connection malformed higher precedence (header precedence, multiple headers, multiple IPs)",
			headers: map[string]string{
				"CF-Connecting-IP": "mistake",
				"X-Forwarded-For":  "1.2.3.4     ,    5.6.7.8  ",
			},
			remote:   "192.168.0.1:999",
			expected: "1.2.3.4",
		},
		{
			name: "proxy connection (multiple IPs)",
			headers: map[string]string{
				"X-Forwarded-For": "1.2.3.4, 5.6.7.8",
			},
			remote:   "192.168.0.1:999",
			expected: "1.2.3.4",
		},
		{
			name:     "spoofing attempt (private IP in header)",
			headers:  map[string]string{"CF-Connecting-IP": "10.0.0.55"},
			remote:   "192.168.0.1:999",
			expected: "192.168.0.1",
		},
		{
			name:     "direct connection invalid IP (no headers)",
			headers:  map[string]string{},
			remote:   "500.500.600.500",
			expected: "",
		},
		{
			name:     "direct connection not-an-ip IP (ho headers), for middleware to process",
			headers:  map[string]string{},
			remote:   "mistake",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// test here
			req, _ := http.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remote

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := getProxyClientIP(req)

			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
