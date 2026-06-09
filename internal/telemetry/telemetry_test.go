package telemetry

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

func TestTelemetryDisabled(t *testing.T) {
	provider, err := Init(context.Background(), Config{Enabled: false})
	if err != nil {
		t.Fatalf("no se esperaba error al deshabilitar telemetría: %v", err)
	}
	if provider == nil {
		t.Fatal("se esperaba provider no nulo")
	}
	if Enabled() {
		t.Fatal("telemetría debería estar deshabilitada")
	}
}

func TestTelemetryStdoutSpan(t *testing.T) {
	provider, err := Init(context.Background(), Config{
		Enabled:        true,
		ServiceName:    "sentinelops-test",
		ServiceVersion: "test",
		Environment:    "test",
		Exporter:       "stdout",
		SampleRate:     1,
	})
	if err != nil {
		t.Fatalf("no se pudo inicializar telemetría: %v", err)
	}
	defer provider.Shutdown(context.Background())

	ctx, span := Tracer().Start(context.Background(), "test.operation")
	span.SetAttributes(attribute.String("test.key", "valor"))
	AnnotateSpanWithCorrelation(ContextWithCorrelationID(ctx, "corr-test"))
	span.End()
}

func TestCorrelationHeaders(t *testing.T) {
	ctx := ContextWithCorrelationID(context.Background(), "corr-123")
	header := http.Header{}
	InjectTracingHeaders(ctx, header)
	if got := header.Get(CorrelationIDHeader); got != "corr-123" {
		t.Fatalf("correlation id inesperado: %q", got)
	}
}

func TestNestedSpans(t *testing.T) {
	provider, err := Init(context.Background(), Config{Enabled: true, Exporter: "stdout", SampleRate: 1})
	if err != nil {
		t.Fatalf("no se pudo inicializar telemetría: %v", err)
	}

	ctx, parent := StartSessionSpan(context.Background(), "tcp", "sess-test", "127.0.0.1:12345")
	_, child := StartCommandSpan(ctx, "status", "sess-test", "student")
	child.End()
	parent.End()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := provider.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("error cerrando telemetría: %v", err)
	}
}
