// server/internal/shared/logger/noop_test.go
package logger_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kilip/opus/server/internal/shared/logger"
)

// TestNoopLogger verifies that NoopLogger complies with the Logger interface
// and all its methods can be invoked safely without panic or output.
func TestNoopLogger(t *testing.T) {
	var l logger.Logger = &logger.NoopLogger{}
	l.Debug("test")
	l.InfoCtx(context.Background(), "test")
	l.Error("test error", errors.New("sentinel"))

	child := l.With(logger.String("k", "v"))
	if child != l {
		t.Error("expected With to return same NoopLogger instance")
	}
}
