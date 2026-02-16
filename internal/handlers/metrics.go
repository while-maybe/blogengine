package handlers

import (
	"blogengine/internal/middleware"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// HandleMetrics returns JSON statistics about memory usage
func (h *BlogHandler) HandleMetrics() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// force GC to get metrics
		// runtime.GC()

		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		var m runtime.MemStats
		// force Go to read the current memory state
		runtime.ReadMemStats(&m)

		uptime := time.Since(h.StartTime).Truncate(time.Second)

		stats := struct {
			Alloc        string                    `json:"allocated_heap_mb"`  // Active objects in heap
			TotalAlloc   string                    `json:"total_alloc_mb"`     // Cumulative allocs (shows churn)
			Sys          string                    `json:"system_obtained_mb"` // Total RAM asked from OS
			NumGC        uint32                    `json:"gc_cycles"`          // Number of garbage collections
			Goroutines   int                       `json:"goroutines"`         // Active "threads"
			Cores        int                       `json:"cpu_cores"`          // Hardware available
			TopCountries []*middleware.CountryStat `json:"top_countries"`
			Uptime       string                    `json:"uptime"`
			ServerTime   time.Time                 `json:"server_time"`
		}{
			Alloc:        bToMb(m.Alloc),
			TotalAlloc:   bToMb(m.TotalAlloc),
			Sys:          bToMb(m.Sys),
			NumGC:        m.NumGC,
			Goroutines:   runtime.NumGoroutine(),
			Cores:        runtime.NumCPU(),
			TopCountries: h.GeoStats.GetTopCountries(20),
			Uptime:       uptime.String(),
			ServerTime:   time.Now().UTC().Truncate(time.Millisecond),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})
}

// Helper to format bytes to MB string
func bToMb(b uint64) string {
	mb := float64(b) / 1024 / 1024
	return fmt.Sprintf("%.2f MB", mb)
}
