package log

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

const (
	keyTraceID = "trace_id"
)

// TraceIDFromContext 从 context 中提取 traceID
func TraceIDFromContext(ctx context.Context) string {
	var traceID string
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().TraceID().IsValid() {
		traceID = span.SpanContext().TraceID().String()
	}

	return traceID
}
