package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"
)

type GeoStats struct {
	mu        sync.RWMutex
	visitors  map[string]time.Time
	countries map[string]int
}

func NewGeoStats(ctx context.Context) *GeoStats {
	g := &GeoStats{
		visitors:  make(map[string]time.Time),
		countries: make(map[string]int),
	}

	go g.backgroundCleanup(ctx)
	return g
}

const validity = 24 * time.Hour
const cleanupFrequency = 15 * time.Minute

func (g *GeoStats) backgroundCleanup(ctx context.Context) {
	ticker := time.NewTicker(cleanupFrequency)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.cleanup()
		}
	}
}

func (g *GeoStats) cleanup() {
	cutOff := time.Now().UTC().Add(-validity)

	g.mu.Lock()
	defer g.mu.Unlock()

	for ip, lastSeen := range g.visitors {
		if lastSeen.Before(cutOff) {
			delete(g.visitors, ip)
		}
	}
}

func (g *GeoStats) Record(ip, countryCode string) {
	if countryCode == "" {
		countryCode = "XX"
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// if does not exist
	if _, ok := g.visitors[ip]; !ok {
		g.countries[countryCode]++
	}

	// update lastSeen time
	g.visitors[ip] = time.Now().UTC()
}

type CountryStat struct {
	Code  string `json:"code"`
	Count int    `json:"count"`
}

func (g *GeoStats) GetTopCountries(n int) []*CountryStat {
	if n < 1 {
		return nil
	}

	topN := make([]*CountryStat, 0, n)

	g.mu.RLock()
	for code, count := range g.countries {
		topN = append(topN, &CountryStat{Code: code, Count: count})
	}
	g.mu.RUnlock()

	slices.SortStableFunc(topN, func(a, b *CountryStat) int {
		switch {
		case a.Count < b.Count:
			return -1
		case a.Count > b.Count:
			return 1
		default:
			return 0
		}
	})

	n = min(n, len(topN))
	return topN[:n]
}

func (g *GeoStats) Middleware(logger *slog.Logger) Middleware {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			go func() {
				// extract CloudFlare country code
				code := r.Header.Get("CF-IPCountry")

				ip := getProxyClientIP(r)

				g.Record(ip, code)
				logger.Info("geo", "ip", ip, "code", code)
			}()
			next.ServeHTTP(w, r)
		})
	}
}
