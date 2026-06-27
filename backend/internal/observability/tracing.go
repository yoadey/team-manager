// Package observability wires optional OpenTelemetry tracing and Sentry-based
// backend error tracking. Both are opt-in: tracing activates only when
// OTEL_EXPORTER_OTLP_ENDPOINT is set, error tracking only when a Sentry DSN is
// provided. When disabled they return no-op shutdown/flush functions, so local
// development and tests need no collector or DSN.
package observability

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// InitTracer configures the global OpenTelemetry tracer provider with an
// OTLP/HTTP exporter (endpoint and options read from the standard OTEL_* env
// vars). It is a no-op when OTEL_EXPORTER_OTLP_ENDPOINT is unset. The returned
// function flushes and shuts the provider down and must be called on exit.
func InitTracer(ctx context.Context, serviceName, version string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return noop, nil
	}

	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return noop, fmt.Errorf("observability.InitTracer: exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewSchemaless(
			attribute.String("service.name", serviceName),
			attribute.String("service.version", version),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	return tp.Shutdown, nil
}
