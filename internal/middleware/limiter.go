package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	ips          map[string]*client
	mu           sync.Mutex
	rate         rate.Limit
	burst        int
	trustedProxy bool
}

var (
	ErrInvalidIP = errors.New("invalid IP")
)

func NewIPRateLimiter(ctx context.Context, rps, burst int, trustedProxy bool) *IPRateLimiter {
	l := &IPRateLimiter{
		ips:          make(map[string]*client),
		rate:         rate.Limit(rps),
		burst:        burst,
		trustedProxy: trustedProxy,
	}

	// cleanup stale entries
	go l.backgroundCleanup(ctx)
	return l
}

func (i *IPRateLimiter) backgroundCleanup(ctx context.Context) {
	cleanupFrequency := 1 * time.Minute

	ticker := time.NewTicker(cleanupFrequency)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			i.cleanup()
		}
	}
}

func (i *IPRateLimiter) cleanup() {
	inactiveLimit := 3 * time.Minute

	i.mu.Lock()
	defer i.mu.Unlock()

	for ip, client := range i.ips {
		if time.Since(client.lastSeen) > inactiveLimit {
			delete(i.ips, ip)
		}
	}
}

func (i *IPRateLimiter) getLimiter(ip string) (*rate.Limiter, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, ErrInvalidIP
	}
	canonicalIP := parsedIP.String()

	i.mu.Lock()
	defer i.mu.Unlock()

	c, ok := i.ips[canonicalIP]
	if !ok {
		limiter := rate.NewLimiter(i.rate, i.burst)
		// and a new client then add it to the ips map
		c = &client{
			limiter:  limiter,
			lastSeen: time.Now().UTC(),
		}
		i.ips[canonicalIP] = c
		return limiter, nil
	}

	c.lastSeen = time.Now().UTC()
	return c.limiter, nil
}

var httpInterestingHeaders = []string{
	"CF-Connecting-IP",
	"X-Forwarded-For",
	"X-Real-IP",
}

type ipClientGetter func(r *http.Request) string

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

func getClientIPFactory(trustedProxy bool) ipClientGetter {
	if trustedProxy {
		return getProxyClientIP
	}
	return getDirectClientIPValidated
}

func (i *IPRateLimiter) Middleware(logger *slog.Logger) Middleware {
	getClientIP := getClientIPFactory(i.trustedProxy)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// grab the source ip address
			ip := getClientIP(r)

			limiter, err := i.getLimiter(ip)
			if err != nil {
				http.Error(w, "invalid ip address", http.StatusBadRequest)
				return
			}

			if !limiter.Allow() {
				// Peek at when next token available (without consuming)
				reservation := limiter.Reserve()
				delay := reservation.Delay()
				reservation.Cancel() // Don't consume

				retrySeconds := int(delay.Seconds())
				retrySeconds = max(1, retrySeconds)

				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(i.burst))
				w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
				w.Header().Set("X-RateLimit-Remaining", "0")

				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// X-RateLimit-Limit: 100           # Maximum requests allowed
// X-RateLimit-Remaining: 42        # Requests remaining in window
// Retry-After: 12                  # Seconds until retry (when limited)
// X-RateLimit-Reset: 1704672000    # Unix timestamp when limit resets
