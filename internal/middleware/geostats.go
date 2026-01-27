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
	mu               sync.RWMutex
	visitors         map[string]time.Time
	countries        map[string]int
	validity         time.Duration
	cleanupFrequency time.Duration
}

func NewGeoStats(ctx context.Context) *GeoStats {

	g := &GeoStats{
		visitors:         make(map[string]time.Time),
		countries:        make(map[string]int),
		validity:         24 * time.Hour,
		cleanupFrequency: 15 * time.Minute,
	}

	go g.backgroundCleanup(ctx)
	return g
}

func (g *GeoStats) backgroundCleanup(ctx context.Context) {
	ticker := time.NewTicker(g.cleanupFrequency)
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
	cutOff := time.Now().UTC().Add(-g.validity)

	g.mu.Lock()
	defer g.mu.Unlock()

	for ip, lastSeen := range g.visitors {
		if lastSeen.Before(cutOff) {
			delete(g.visitors, ip)
		}
	}
}

func (g *GeoStats) Record(ip, countryCode string) {
	if ip == "" {
		return
	}

	if countryCode == "" {
		countryCode = "XX"
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// if does not exist
	if lastSeen, ok := g.visitors[ip]; !ok || time.Since(lastSeen) > g.validity {
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
		return []*CountryStat{}
	}

	topN := make([]*CountryStat, 0, n)

	g.mu.RLock()
	if len(g.countries) == 0 {
		g.mu.RUnlock()
		return []*CountryStat{}
	}

	for code, count := range g.countries {
		topN = append(topN, &CountryStat{Code: code, Count: count})
	}
	g.mu.RUnlock()

	slices.SortStableFunc(topN, func(a, b *CountryStat) int {
		// descending order
		return b.Count - a.Count
	})

	n = min(n, len(topN))
	return topN[:n]
}

func (g *GeoStats) Middleware(logger *slog.Logger) Middleware {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			code := r.Header.Get("CF-IPCountry")
			ip := getProxyClientIP(r)

			go func() {
				// extract CloudFlare country code
				g.Record(ip, code)
				logger.Info("geo", "ip", ip, "code", code)
			}()
			next.ServeHTTP(w, r)
		})
	}
}
