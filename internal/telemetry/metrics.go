package telemetry

import (
	"context"
	"fmt"

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
	}, nil
}

// RecordPostsLoaded updates the gauge with current post count
func (m *Metrics) RecordPostsLoaded(ctx context.Context, count int) {
	m.PostsLoadedTotal.Add(ctx, int64(count))
}
