package telemetry

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	// CorrelationIDHeader es el header HTTP usado para correlacionar solicitudes.
	CorrelationIDHeader = "X-Correlation-ID"
	// TraceIDHeader expone el trace ID para depuración controlada.
	TraceIDHeader = "X-Trace-ID"
	// CorrelationBaggageKey es la clave propagada en baggage.
	CorrelationBaggageKey = "correlation.id"
)

// NewCorrelationID genera un identificador de correlación.
func NewCorrelationID() string {
	return uuid.NewString()
}

// ContextWithCorrelationID agrega el correlation ID al contexto.
func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	if correlationID == "" {
		correlationID = NewCorrelationID()
	}
	member, err := baggage.NewMember(CorrelationBaggageKey, correlationID)
	if err != nil {
		return ctx
	}
	bag, err := baggage.New(member)
	if err != nil {
		return ctx
	}
	return baggage.ContextWithBaggage(ctx, bag)
}

// ExtractCorrelationID obtiene el correlation ID desde baggage.
func ExtractCorrelationID(ctx context.Context) string {
	member := baggage.FromContext(ctx).Member(CorrelationBaggageKey)
	return member.Value()
}

// InjectTracingHeaders propaga traceparent, baggage y correlation ID en una solicitud saliente.
func InjectTracingHeaders(ctx context.Context, header http.Header) {
	otelPropagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otelPropagator.Inject(ctx, propagation.HeaderCarrier(header))
	if correlationID := ExtractCorrelationID(ctx); correlationID != "" {
		header.Set(CorrelationIDHeader, correlationID)
	}
}

// AnnotateSpanWithCorrelation agrega el correlation ID al span activo.
func AnnotateSpanWithCorrelation(ctx context.Context) {
	correlationID := ExtractCorrelationID(ctx)
	if correlationID == "" {
		return
	}
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("correlation.id", correlationID))
}
