package telemetry

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Setup initializes OpenTelemetry tracing and metrics.
// If OTEL_EXPORTER_OTLP_ENDPOINT is set, exports to OTLP (gRPC).
// Otherwise, traces are written to stdout for local development.
func Setup(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	traceShutdown, err := setupTracer(ctx, res)
	if err != nil {
		return nil, err
	}

	metricShutdown, err := setupMeter(ctx, res)
	if err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		if err := traceShutdown(ctx); err != nil {
			return err
		}
		return metricShutdown(ctx)
	}, nil
}

func setupTracer(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	var exporter sdktrace.SpanExporter
	var err error

	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		exporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(endpoint))
	} else {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

func setupMeter(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	var reader sdkmetric.Reader

	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpoint(endpoint))
		if err != nil {
			return nil, err
		}
		reader = sdkmetric.NewPeriodicReader(exporter)
	} else {
		reader = sdkmetric.NewManualReader()
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)
	otel.SetMeterProvider(mp)

	return mp.Shutdown, nil
}
