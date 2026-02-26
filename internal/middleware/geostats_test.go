package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"go.opentelemetry.io/otel/trace/noop"
)

func TestGetTopCountries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		n            int
		countryStats map[string]int
		want         []*CountryStat
	}{
		{name: "map empty, n=5", n: 5,
			countryStats: map[string]int{},
			want:         []*CountryStat{},
		},
		{name: "5 entries, n=3", n: 3,
			countryStats: map[string]int{"AA": 10, "BB": 20, "CC": 30, "DD": 40, "EE": 100},
			want:         []*CountryStat{{Code: "EE", Count: 100}, {Code: "DD", Count: 40}, {Code: "CC", Count: 30}},
		},
		{name: "1 entry, n=0", n: 0,
			countryStats: map[string]int{"AA": 10},
			want:         []*CountryStat{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gs := NewGeoStats(t.Context())

			gs.mu.Lock()
			gs.countries = tt.countryStats
			gs.mu.Unlock()

			result := gs.GetTopCountries(tt.n)

			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("topN countries mismatch: want: %v, got: %v", tt.want, result)
			}
		})
	}
}

func TestRecord(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		setupVisitors map[string]time.Time
		setupCount    map[string]int
		ip            string
		code          string
		wantCode      string
		wantCount     int
	}{
		{name: "new visitor, increments count",
			setupVisitors: map[string]time.Time{},
			setupCount:    map[string]int{},
			ip:            "1.1.1.1",
			code:          "GB",
			wantCode:      "GB",
			wantCount:     1,
		},
		{name: "new visitor, no country code, increments XX count",
			setupVisitors: map[string]time.Time{},
			setupCount:    map[string]int{},
			ip:            "1.1.1.1",
			code:          "",
			wantCode:      "XX",
			wantCount:     1,
		},
		{name: "new visitor, increments count",
			setupVisitors: map[string]time.Time{"1.1.1.1": time.Now()},
			setupCount:    map[string]int{"GB": 10},
			ip:            "2.2.2.2",
			code:          "GB",
			wantCode:      "GB",
			wantCount:     11,
		},
		// happens if visit was less than 24h ago
		{name: "existing visitor, does not increments count",
			setupVisitors: map[string]time.Time{"1.1.1.1": time.Now()},
			setupCount:    map[string]int{"GB": 10},
			ip:            "1.1.1.1",
			code:          "GB",
			wantCode:      "GB",
			wantCount:     10,
		},
		// happens if visit was more than 24h ago - should not even be in map
		{name: "existing visitor, increments count",
			setupVisitors: map[string]time.Time{"1.1.1.1": time.Now().Add(-25 * time.Hour)},
			setupCount:    map[string]int{"GB": 10},
			ip:            "1.1.1.1",
			code:          "GB",
			wantCode:      "GB",
			wantCount:     11,
		},
		// happens if visit was more than 24h ago - should not even be in map
		{name: "new visitor, no IP, discard",
			setupVisitors: map[string]time.Time{"1.1.1.1": time.Now().Add(-25 * time.Hour)},
			setupCount:    map[string]int{"GB": 10},
			ip:            "",
			code:          "XX",
			wantCode:      "",
			wantCount:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gs := NewGeoStats(t.Context())

			// populate starting data if it exists
			gs.mu.Lock()
			if tt.setupVisitors != nil {
				gs.visitors = tt.setupVisitors
			}
			if tt.setupCount != nil {
				gs.countries = tt.setupCount
			}
			gs.mu.Unlock()

			gs.Record(tt.ip, tt.code)

			// verify
			gs.mu.RLock()
			defer gs.mu.RUnlock()

			if tt.ip == "" {
				// empty string should not be added as key
				if _, ok := gs.visitors[""]; ok {
					t.Errorf("Security fail: Empty IP key was created in visitors map")
				}

				// country count cannot change
				if tt.wantCode != "" {
					got := gs.countries[tt.wantCode]
					if got != tt.wantCount {
						t.Errorf("Count changed for invalid IP: want %d, got %d", tt.wantCount, got)
					}
				}

				// nothing else to test if no IP was given
				return
			}

			lastSeen, ok := gs.visitors[tt.ip]

			if !ok {
				t.Errorf("visitor IP %q was not recorded in map", tt.ip)
			}

			if time.Since(lastSeen) > 1*time.Second {
				t.Errorf("visitor timestamp was not updated to now")
			}

			if gotCount, ok := gs.countries[tt.wantCode]; !ok || gotCount != tt.wantCount {
				t.Errorf("count for %s: expected %d, got %d", tt.wantCode, tt.wantCount, gotCount)
			}
		})
	}
}

func TestBackgroundCleanup(t *testing.T) {
	// t.Parallel()
	tests := []struct {
		name           string
		setupVisitors  map[string]time.Duration
		setupCountries map[string]int
		wantVisitors   []string
		wantCountries  map[string]int
	}{
		{name: "empty gs, does nothing",
			setupVisitors:  map[string]time.Duration{},
			setupCountries: map[string]int{},
			wantVisitors:   []string{},
			wantCountries:  map[string]int{},
		},
		{name: "1 stale, 0 recent, leaves 0",
			setupVisitors:  map[string]time.Duration{"1.1.1.1": -36 * time.Hour},
			setupCountries: map[string]int{"GB": 1},
			wantVisitors:   []string{},
			wantCountries:  map[string]int{"GB": 1},
		},
		{name: "1 stale, 1 recent, leaves 1",
			setupVisitors:  map[string]time.Duration{"1.1.1.1": -36 * time.Hour, "2.2.2.2": -12 * time.Hour},
			setupCountries: map[string]int{"GB": 1, "US": 1},
			wantVisitors:   []string{"2.2.2.2"},
			wantCountries:  map[string]int{"GB": 1, "US": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gs *GeoStats
			// do time bubble test -
			synctest.Test(t, func(t *testing.T) {
				// Advance time to approximately now
				gs = NewGeoStats(t.Context())

				// inject starting data
				gs.mu.Lock()
				// we have to use durations as clock starts 2000-01-01 inside the bubble
				for ip, dur := range tt.setupVisitors {
					gs.visitors[ip] = time.Now().Add(dur)
				}

				if tt.setupCountries != nil {
					gs.countries = tt.setupCountries
				}
				gs.mu.Unlock()

				// sleeping 1h makes .backgroundCleanup() ticker run 4 times running .cleanup() 4 times
				time.Sleep(16 * time.Minute)
			})

			gs.mu.RLock()
			defer gs.mu.RUnlock()

			if !reflect.DeepEqual(tt.wantCountries, gs.countries) {
				t.Errorf("countries mistmatch: want %v, got %v", tt.wantCountries, gs.countries)
			}

			if len(gs.visitors) != len(tt.wantVisitors) {
				t.Errorf("visitor count mismatch: want %d, got %d", len(tt.wantVisitors), len(gs.visitors))
			}

			for _, ip := range tt.wantVisitors {
				if _, ok := gs.visitors[ip]; !ok {
					t.Errorf("expected visitor %s to exist, but it was removed", ip)
				}
			}
		})
	}
}

func TestGeoStats_Middleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		wantCode   string
		wantCount  int
	}{
		{
			name: "normal cloudflare request",
			headers: map[string]string{
				"CF-IPCountry":     "NZ",
				"CF-Connecting-IP": "100.100.100.100",
			},
			remoteAddr: "10.10.10.10:55555",
			wantCode:   "NZ",
			wantCount:  1,
		},
		{
			name:       "direct connection (no headers)",
			remoteAddr: "192.168.5.50:1234",
			wantCode:   "XX",
			wantCount:  1,
		},
		{
			name:       "invalid IP addr",
			remoteAddr: "mistake",
			wantCode:   "",
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := slog.New(slog.DiscardHandler)
			gs := NewGeoStats(t.Context())

			// handler does nothing
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

			middlewareToTest := gs.Middleware(logger, noop.NewTracerProvider().Tracer(""))(nextHandler)

			req, _ := http.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			recorder := httptest.NewRecorder()

			middlewareToTest.ServeHTTP(recorder, req)

			if tt.wantCount == 0 {
				time.Sleep(50 * time.Millisecond)

				stats := gs.GetTopCountries(10)
				// Look for code, if found, fail
				for _, s := range stats {
					if s.Code == tt.wantCode {
						t.Errorf("Security fail: Found stats for %q when expecting ignore", tt.wantCode)
					}
				}
				return
			}

			// this is done async so things might take a little while, we introduce a little delay
			var gotData bool
			for range 50 {
				stats := gs.GetTopCountries(10)
				for _, stat := range stats {
					if stat.Code == tt.wantCode && stat.Count == tt.wantCount {
						gotData = true
						break
					}
				}
				time.Sleep(20 * time.Millisecond)
			}

			if !gotData {
				t.Errorf("Stats not updated asynchronously. Want code=%s count=%d", tt.wantCode, tt.wantCount)
			}
		})
	}
}
