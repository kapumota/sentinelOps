package telemetry

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "sentinelops"

// Config define la configuración de OpenTelemetry para SentinelOps.
type Config struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	Environment    string
	Exporter       string
	Endpoint       string
	Insecure       bool
	SampleRate     float64
}

// Provider encapsula el proveedor de trazas para poder cerrarlo de forma ordenada.
type Provider struct {
	enabled bool
	tracer  trace.Tracer
	tp      *sdktrace.TracerProvider
}

var globalProvider = &Provider{tracer: otel.Tracer(tracerName)}

// Init configura el proveedor global de trazas. Si está deshabilitado, mantiene el proveedor no-op.
func Init(ctx context.Context, cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		globalProvider = &Provider{enabled: false, tracer: otel.Tracer(tracerName)}
		return globalProvider, nil
	}

	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		serviceName = "sentinelops"
	}
	serviceVersion := strings.TrimSpace(cfg.ServiceVersion)
	if serviceVersion == "" {
		serviceVersion = "dev"
	}
	environment := strings.TrimSpace(cfg.Environment)
	if environment == "" {
		environment = "dev"
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("service.version", serviceVersion),
			attribute.String("deployment.environment", environment),
			attribute.String("host.name", hostname()),
		),
		resource.WithProcess(),
		resource.WithOS(),
	)
	if err != nil {
		return nil, fmt.Errorf("crear recurso de telemetría: %w", err)
	}

	exporter, err := newExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithExportTimeout(10*time.Second),
		),
		sdktrace.WithSampler(newSampler(cfg.SampleRate)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	provider := &Provider{
		enabled: true,
		tracer:  tp.Tracer(tracerName, trace.WithInstrumentationVersion(serviceVersion)),
		tp:      tp,
	}
	globalProvider = provider
	return provider, nil
}

// Tracer devuelve el tracer global de SentinelOps.
func Tracer() trace.Tracer {
	if globalProvider == nil || globalProvider.tracer == nil {
		return otel.Tracer(tracerName)
	}
	return globalProvider.tracer
}

// Enabled indica si la telemetría está exportando trazas.
func Enabled() bool {
	return globalProvider != nil && globalProvider.enabled
}

// Shutdown vacía y cierra el proveedor de trazas.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil || p.tp == nil {
		return nil
	}
	return p.tp.Shutdown(ctx)
}

func newExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	exporter := strings.TrimSpace(strings.ToLower(cfg.Exporter))
	if exporter == "" {
		exporter = "stdout"
	}

	switch exporter {
	case "stdout":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "jaeger", "otlp-grpc":
		endpoint := strings.TrimSpace(cfg.Endpoint)
		if endpoint == "" {
			endpoint = "localhost:4317"
		}
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		return otlptracegrpc.New(ctx, opts...)
	case "otlp-http":
		endpoint := strings.TrimSpace(cfg.Endpoint)
		if endpoint == "" {
			endpoint = "localhost:4318"
		}
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		return otlptracehttp.New(ctx, opts...)
	default:
		return nil, fmt.Errorf("exportador de telemetría no soportado: %s", cfg.Exporter)
	}
}

func newSampler(rate float64) sdktrace.Sampler {
	switch {
	case rate <= 0:
		return sdktrace.NeverSample()
	case rate >= 1:
		return sdktrace.AlwaysSample()
	default:
		return sdktrace.TraceIDRatioBased(rate)
	}
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		return "desconocido"
	}
	return name
}
