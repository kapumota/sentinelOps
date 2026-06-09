package telemetry

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware instrumenta handlers HTTP con trazas y correlation IDs.
func HTTPMiddleware(operation string, next http.Handler) http.Handler {
	return otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = NewCorrelationID()
		}

		ctx := ContextWithCorrelationID(r.Context(), correlationID)
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(
			attribute.String("correlation.id", correlationID),
			attribute.String("http.request_id", correlationID),
		)

		w.Header().Set(CorrelationIDHeader, correlationID)
		if span.SpanContext().IsValid() {
			w.Header().Set(TraceIDHeader, span.SpanContext().TraceID().String())
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	}), operation)
}
