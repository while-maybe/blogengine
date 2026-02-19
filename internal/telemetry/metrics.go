package telemetry

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/metric"
)

// Metrics holds all the metric instruments for the blog engine
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal   metric.Int64Counter
	HTTPRequestDuration metric.Float64Histogram
	HTTPActiveRequests  metric.Int64UpDownCounter
	// Blog specific
	PostsLoadedTotal metric.Int64UpDownCounter
	PostViewsTotal   metric.Int64Counter
	CacheHitsTotal   metric.Int64Counter
	CacheMissesTotal metric.Int64Counter
	// limiter
	RateLimitHitsTotal metric.Int64Counter
	// assets
	AssetRequestsTotal metric.Int64Counter
	// middlewares
	AuthWorkDuration metric.Float64Histogram
	Uptime           metric.Float64ObservableGauge
	HeapAlloc        metric.Float64ObservableGauge
	GoRoutines       metric.Int64ObservableGauge
}

func NewMetrics(meter metric.Meter) (*Metrics, error) {
	httpRequestsTotal, err := meter.Int64Counter(
		"http_requests",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http_requests: %w", err)
	}

	httpRequestDuration, err := meter.Float64Histogram(
		"http_request_duration",
		metric.WithDescription("HTTP request latency in ms"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http_request_duration_seconds: %w", err)
	}

	httpActiveRequests, err := meter.Int64UpDownCounter(
		"http_active_requests",
		metric.WithDescription("Number of in-flight requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http_active_requests: %w", err)
	}

	postsLoadedTotal, err := meter.Int64UpDownCounter(
		"posts_loaded",
		metric.WithDescription("Number of posts currently loaded in memory"),
		metric.WithUnit("{post}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create posts_loaded: %w", err)
	}

	postViewsTotal, err := meter.Int64Counter(
		"post_views",
		metric.WithDescription("Total number of post views"),
		metric.WithUnit("{view}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create post_views: %w", err)
	}

	cacheHitsTotal, err := meter.Int64Counter(
		"cache_hits",
		metric.WithDescription("Number of cache hits"),
		metric.WithUnit("{hit}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache_hits: %w", err)
	}

	cacheMissesTotal, err := meter.Int64Counter(
		"cache_misses",
		metric.WithDescription("Number of cache misses"),
		metric.WithUnit("{miss}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache_misses: %w", err)
	}

	rateLimitHitsTotal, err := meter.Int64Counter(
		"rate_limit_hits",
		metric.WithDescription("Number of rate limiter blocked requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create rate_limit_hits: %w", err)
	}

	assetRequestsTotal, err := meter.Int64Counter(
		"asset_requests",
		metric.WithDescription("Total number of assets requested"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset_requests_total: %w", err)
	}

	authWorkDuration, err := meter.Float64Histogram(
		"auth_work_duration",
		metric.WithDescription("real time spent on DB/Bcrypt"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth_work_duration: %w", err)
	}

	uptime, err := meter.Float64ObservableGauge("process_uptime_seconds")
	if err != nil {
		return nil, fmt.Errorf("failed to create process_uptime_seconds: %w", err)
	}

	heap, err := meter.Float64ObservableGauge("process_memory_heap_alloc_mb")
	if err != nil {
		return nil, fmt.Errorf("failed to create process_memory_heap_alloc: %w", err)
	}
	goroutines, err := meter.Int64ObservableGauge(
		"process_goroutines",
		metric.WithDescription("Active goroutines"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create process_goroutines: %w", err)
	}

	startTime := time.Now().UTC()
	_, err = meter.RegisterCallback(func(_ context.Context, obs metric.Observer) error {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		obs.ObserveFloat64(uptime, time.Since(startTime).Seconds())
		obs.ObserveFloat64(heap, float64(m.Alloc)/1024/1024)
		obs.ObserveInt64(goroutines, int64(runtime.NumGoroutine()))

		return nil
	}, uptime, heap, goroutines)

	if err != nil {
		return nil, fmt.Errorf("failed to register metrics callback: %w", err)
	}

	return &Metrics{
		HTTPRequestsTotal:   httpRequestsTotal,
		HTTPRequestDuration: httpRequestDuration,
		HTTPActiveRequests:  httpActiveRequests,
		PostsLoadedTotal:    postsLoadedTotal,
		PostViewsTotal:      postViewsTotal,
		CacheHitsTotal:      cacheHitsTotal,
		CacheMissesTotal:    cacheMissesTotal,
		RateLimitHitsTotal:  rateLimitHitsTotal,
		AssetRequestsTotal:  assetRequestsTotal,
		AuthWorkDuration:    authWorkDuration,
		Uptime:              uptime,
		HeapAlloc:           heap,
		GoRoutines:          goroutines,
	}, nil
}

// RecordPostsLoaded updates the gauge with current post count
func (m *Metrics) RecordPostsLoaded(ctx context.Context, count int) {
	m.PostsLoadedTotal.Add(ctx, int64(count))
}
