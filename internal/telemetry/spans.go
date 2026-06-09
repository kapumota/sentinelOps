package telemetry

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartSessionSpan crea un span raíz para una sesión TCP o SSH.
func StartSessionSpan(ctx context.Context, transport string, sessionID string, remoteAddr string) (context.Context, trace.Span) {
	return Tracer().Start(ctx, fmt.Sprintf("%s.session", transport),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("session.id", sessionID),
			attribute.String("network.transport", "tcp"),
			attribute.String("network.protocol.name", transport),
			attribute.String("net.peer.address", remoteAddr),
		),
	)
}

// StartAuthSpan crea un span para autenticación.
func StartAuthSpan(ctx context.Context, method string, username string, remoteAddr string) (context.Context, trace.Span) {
	return Tracer().Start(ctx, "auth.authenticate",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("auth.method", method),
			attribute.String("enduser.id", username),
			attribute.String("net.peer.address", remoteAddr),
		),
	)
}

// StartValidationSpan crea un span para validación de entrada.
func StartValidationSpan(ctx context.Context, validatorType string, inputLength int) (context.Context, trace.Span) {
	return Tracer().Start(ctx, "security.validate_input",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("validator.type", validatorType),
			attribute.Int("input.length", inputLength),
		),
	)
}

// StartCommandSpan crea un span para comandos interactivos.
func StartCommandSpan(ctx context.Context, command string, sessionID string, username string) (context.Context, trace.Span) {
	return Tracer().Start(ctx, "command.execute",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("command.name", command),
			attribute.String("session.id", sessionID),
			attribute.String("enduser.id", username),
		),
	)
}

// StartForwardingSpan crea un span para túneles SSH.
func StartForwardingSpan(ctx context.Context, direction string, source string, destination string, sessionID string) (context.Context, trace.Span) {
	return Tracer().Start(ctx, "ssh.forwarding",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("ssh.forward.direction", direction),
			attribute.String("ssh.forward.source", source),
			attribute.String("ssh.forward.destination", destination),
			attribute.String("session.id", sessionID),
		),
	)
}

// SetSpanError marca un span como error y registra el error.
func SetSpanError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetSpanOK marca un span como correcto.
func SetSpanOK(span trace.Span, message string) {
	if span == nil {
		return
	}
	span.SetStatus(codes.Ok, message)
}

// PeerParts separa host y puerto para atributos de red.
func PeerParts(addr string) (string, string) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, ""
	}
	return host, port
}

// AddDuration agrega duración medida manualmente al span.
func AddDuration(span trace.Span, started time.Time, attrName string) {
	if span == nil || attrName == "" {
		return
	}
	span.SetAttributes(attribute.Int64(attrName, time.Since(started).Milliseconds()))
}
