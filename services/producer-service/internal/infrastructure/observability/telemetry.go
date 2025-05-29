package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TelemetryConfig holds OpenTelemetry configuration
type TelemetryConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string
	Insecure       bool
	SampleRate     float64
}

// TelemetryProvider holds OpenTelemetry providers
type TelemetryProvider struct {
	traceProvider  *sdktrace.TracerProvider
	metricProvider *sdkmetric.MeterProvider
	config         TelemetryConfig
}

// NewTelemetryProvider creates a new OpenTelemetry provider
func NewTelemetryProvider(config TelemetryConfig) (*TelemetryProvider, error) {
	ctx := context.Background()

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize trace provider
	traceProvider, err := initTraceProvider(ctx, res, config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize trace provider: %w", err)
	}

	// Initialize metric provider
	metricProvider, err := initMetricProvider(ctx, res, config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metric provider: %w", err)
	}

	// Set global providers
	otel.SetTracerProvider(traceProvider)
	otel.SetMeterProvider(metricProvider)

	// Set text map propagator for context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TelemetryProvider{
		traceProvider:  traceProvider,
		metricProvider: metricProvider,
		config:         config,
	}, nil
}

// initTraceProvider initializes the trace provider
func initTraceProvider(ctx context.Context, res *resource.Resource, config TelemetryConfig) (*sdktrace.TracerProvider, error) {
	conn, err := grpc.NewClient(config.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithGRPCConn(conn),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Configure trace provider with batch span processor
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SampleRate)),
	)

	return tp, nil
}

// initMetricProvider initializes the metric provider
func initMetricProvider(ctx context.Context, res *resource.Resource, config TelemetryConfig) (*sdkmetric.MeterProvider, error) {
	conn, err := grpc.NewClient(config.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection for metrics: %w", err)
	}

	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithGRPCConn(conn),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(15*time.Second),
		)),
	)

	return mp, nil
}

// Shutdown gracefully shuts down the telemetry providers
func (tp *TelemetryProvider) Shutdown(ctx context.Context) error {
	var errs []error

	if err := tp.traceProvider.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown trace provider: %w", err))
	}

	if err := tp.metricProvider.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown metric provider: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}

// GetTracer returns a tracer instance
func (tp *TelemetryProvider) GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// GetMeter returns a meter instance
func (tp *TelemetryProvider) GetMeter(name string) metric.Meter {
	return otel.Meter(name)
}
