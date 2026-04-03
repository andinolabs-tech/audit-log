package telemetry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"audit-log/internal/infra/config"
)

// Install configures global TracerProvider, MeterProvider, and B3 propagation.
// When cfg.OTelEnabled is false, only B3 + noop providers are set (no network).
func Install(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {
	otel.SetTextMapPropagator(b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)))

	if !cfg.OTelEnabled {
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		otel.SetMeterProvider(sdkmetric.NewMeterProvider())
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.OTelServiceName),
			semconv.DeploymentEnvironment(cfg.OTelEnvironment),
			semconv.ServiceVersion("0.0.0-dev"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	traceExp, err := otlptrace.New(ctx, otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(cfg.OTelEndpoint),
		otlptracegrpc.WithInsecure(),
	))
	if err != nil {
		return nil, fmt.Errorf("otlp trace exporter: %w", err)
	}

	sampler := buildSampler(cfg)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)
	otel.SetTracerProvider(tp)

	metricExp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.OTelEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("otlp metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp,
			sdkmetric.WithInterval(5*time.Second),
		)),
	)
	otel.SetMeterProvider(mp)

	return func(shutdownCtx context.Context) error {
		return errors.Join(
			mp.Shutdown(shutdownCtx),
			tp.Shutdown(shutdownCtx),
		)
	}, nil
}

func buildSampler(cfg *config.Config) sdktrace.Sampler {
	env := strings.ToLower(cfg.OTelEnvironment)
	if env == "development" || env == "dev" || env == "staging" {
		return sdktrace.AlwaysSample()
	}
	r := cfg.OTelSampleRate
	if r <= 0 {
		r = 0.1
	}
	if r > 1 {
		r = 1
	}
	return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(r))
}
