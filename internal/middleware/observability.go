package middleware

import (
	"blogengine/internal/telemetry"
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func Observability(tracer trace.Tracer, metrics *telemetry.Metrics, logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			traceID := uuid.Must(uuid.NewV7()).String()

			ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.route", r.URL.Path),
					attribute.String("http.user_agent", r.Header.Get("User-Agent")),
					attribute.String("trace.id", traceID),
				),
			)
			defer span.End()

			w.Header().Set("X-Trace-ID", traceID)

			// new logger
			logger := logger.With("trace_id", traceID, "span_id", span.SpanContext().SpanID().String())

			// using a bare string as a ctx key will cause a staticcheck error.
			type contextKey string
			const loggerKey contextKey = "logger"

			// and add to context
			ctx = context.WithValue(ctx, loggerKey, logger)

			// track active requests
			metrics.HTTPActiveRequests.Add(ctx, 1)
			defer metrics.HTTPActiveRequests.Add(ctx, -1)

			start := time.Now()

			// wrapper response writer
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r.WithContext(ctx))
			if wrapped.statusCode >= 400 {
				span.SetStatus(codes.Error, http.StatusText(wrapped.statusCode))
			} else {
				span.SetStatus(codes.Ok, "OK")
			}

			duration := float64(time.Since(start).Milliseconds())

			attrs := []attribute.KeyValue{
				attribute.String("http.method", r.Method),
				attribute.String("http.route", r.URL.Path),
				attribute.Int("http.status_code", wrapped.statusCode),
			}

			metrics.HTTPRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
			metrics.HTTPRequestDuration.Record(ctx, duration, metric.WithAttributes(attrs...))

			span.SetAttributes(
				attribute.Int("http.status_code", wrapped.statusCode),
				attribute.Float64("http.duration_ms", duration),
			)

			logger.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration", duration,
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *responseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
