// server/internal/shared/logger/context.go
package logger

import "context"

type contextKey int

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey contextKey = iota
	// UserIDKey is the context key for user ID.
	UserIDKey
	// TraceIDKey is the context key for OpenTelemetry trace ID.
	TraceIDKey
	// SpanIDKey is the context key for OpenTelemetry span ID.
	SpanIDKey
)

const (
	// FieldRequestID is the log field key for request ID.
	FieldRequestID = "request_id"
	// FieldUserID is the log field key for user ID.
	FieldUserID = "user_id"
	// FieldTraceID is the log field key for trace ID.
	FieldTraceID = "trace_id"
	// FieldSpanID is the log field key for span ID.
	FieldSpanID = "span_id"
)

// WithRequestID injects a request ID into the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestIDKey, id)
}

// WithUserID injects a user ID into the context.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, UserIDKey, id)
}

// WithTraceID injects an OpenTelemetry trace ID into the context.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, TraceIDKey, id)
}

// WithSpanID injects an OpenTelemetry span ID into the context.
func WithSpanID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, SpanIDKey, id)
}

// ExtractFields retrieves all registered tracing fields from the context.
func ExtractFields(ctx context.Context) []Field {
	if ctx == nil {
		return nil
	}
	var fields []Field
	if v, ok := ctx.Value(RequestIDKey).(string); ok && v != "" {
		fields = append(fields, String(FieldRequestID, v))
	}
	if v, ok := ctx.Value(UserIDKey).(string); ok && v != "" {
		fields = append(fields, String(FieldUserID, v))
	}
	if v, ok := ctx.Value(TraceIDKey).(string); ok && v != "" {
		fields = append(fields, String(FieldTraceID, v))
	}
	if v, ok := ctx.Value(SpanIDKey).(string); ok && v != "" {
		fields = append(fields, String(FieldSpanID, v))
	}
	return fields
}
