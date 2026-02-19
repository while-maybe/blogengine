package telemetry

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

type Telemetry struct {
	TraceProvider *sdktrace.TracerProvider
	MeterProvider *sdkmetric.MeterProvider
	Tracer        trace.Tracer
	Meter         metric.Meter
	logger        *slog.Logger
}

// Init sets up OpenTelemetry with both tracing and metrics
func Init(ctx context.Context, serviceName, serviceVersion, env, otelEndpoint string, enabled bool, logger *slog.Logger) (*Telemetry, error) {

	if !enabled {
		logger.Info("telemetry disabled: using no-op providers")
		return &Telemetry{
			TraceProvider: nil,                                      // No provider needed
			MeterProvider: nil,                                      // No provider needed
			Tracer:        tracenoop.NewTracerProvider().Tracer(""), // Dummy Tracer
			Meter:         noop.NewMeterProvider().Meter(""),        // Dummy Meter
			logger:        logger,
		}, nil
	}

	// create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			attribute.String("environment", env),
		),
	)
	if err != nil {
		return nil, err
	}

	// setup tracing
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otelEndpoint),
		otlptracehttp.WithInsecure(),
	)

	// traceExporter, err := otlptracegrpc.New(ctx,
	// 	otlptracegrpc.WithEndpointURL(otelEndpoint),
	// 	otlptracegrpc.WithInsecure(),
	// )
	if err != nil {
		logger.Warn("failed to create trace exporter, traces disabled", "err", err)
		traceExporter = nil
	}

	var traceProvider *sdktrace.TracerProvider
	if traceExporter != nil {
		traceProvider = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(traceExporter),
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
		)
	} else {
		traceProvider = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
		)
	}

	otel.SetTracerProvider(traceProvider)

	// setup metrics
	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(otelEndpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		logger.Warn("failed to create metric exporter, metrics disabled", "err", err)
		metricExporter = nil
	}

	const dataPushInterval = 5 * time.Second
	var meterProvider *sdkmetric.MeterProvider
	if metricExporter != nil {
		meterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			// This tells the app to PUSH every 30-60 seconds
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
				metricExporter,
				sdkmetric.WithInterval(dataPushInterval),
			)),
		)
	} else {
		meterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
		)
	}

	otel.SetMeterProvider(meterProvider)

	// setup ctx propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Telemetry{
		TraceProvider: traceProvider,
		MeterProvider: meterProvider,
		Tracer:        traceProvider.Tracer(serviceName),
		Meter:         meterProvider.Meter(serviceName),
		logger:        logger,
	}, nil
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	// if noop exit now
	if t.TraceProvider == nil && t.MeterProvider == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := t.TraceProvider.Shutdown(ctx); err != nil {
		t.logger.Error("failed to shutdown tracer provider", "err", err)
	}

	if err := t.MeterProvider.Shutdown(ctx); err != nil {
		t.logger.Error("failed to shutdown meter provider", "err", err)
	}

	t.logger.Info("telemetry shutdown complete")
	return nil
}
