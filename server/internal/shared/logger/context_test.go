// server/internal/shared/logger/context_test.go
package logger_test

import (
	"context"
	"testing"

	"github.com/kilip/opus/server/internal/shared/logger"
)

func TestContextTracing(t *testing.T) {
	ctx := context.Background()
	ctx = logger.WithRequestID(ctx, "req-123")
	ctx = logger.WithUserID(ctx, "user-456")
	ctx = logger.WithTraceID(ctx, "trace-789")
	ctx = logger.WithSpanID(ctx, "span-000")

	fields := logger.ExtractFields(ctx)
	if len(fields) != 4 {
		t.Fatalf("expected 4 tracing fields, got %d", len(fields))
	}

	fieldMap := make(map[string]string)
	for _, f := range fields {
		fieldMap[f.Key] = f.Value.String()
	}

	if fieldMap["request_id"] != "req-123" {
		t.Errorf("expected request_id req-123, got %s", fieldMap["request_id"])
	}
	if fieldMap["user_id"] != "user-456" {
		t.Errorf("expected user_id user-456, got %s", fieldMap["user_id"])
	}
	if fieldMap["trace_id"] != "trace-789" {
		t.Errorf("expected trace_id trace-789, got %s", fieldMap["trace_id"])
	}
	if fieldMap["span_id"] != "span-000" {
		t.Errorf("expected span_id span-000, got %s", fieldMap["span_id"])
	}
}
