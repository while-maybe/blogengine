package middleware

import (
	"blogengine/internal/telemetry"
	"net/http"
	"time"
)

func SecureDelay(target time.Duration, metrics *telemetry.Metrics) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			elapsed := time.Since(start)
			metrics.AuthWorkDuration.Record(r.Context(), float64(elapsed.Milliseconds()))

			if remaining := target - elapsed; remaining > 0 {
				timer := time.NewTimer(remaining)
				defer timer.Stop()

				select {
				case <-r.Context().Done():
					return
				case <-timer.C:
					// job done
				}
			}
		})
	}
}
