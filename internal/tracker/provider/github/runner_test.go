package github

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kilip/opus/internal/tracker/domain"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("TEST_HELPER_PROCESS") != "1" {
		return
	}
	mode := os.Getenv("TEST_GH_HELPER_MODE")
	switch mode {
	case "ok":
		_, _ = fmt.Fprint(os.Stdout, `[{"number":1}]`)
		os.Exit(0)
	case "rate_limit":
		_, _ = fmt.Fprintln(os.Stderr, "GraphQL: API Rate Limit exceeded for user")
		os.Exit(1)
	case "error":
		_, _ = fmt.Fprintln(os.Stderr, "not logged into any GitHub hosts")
		os.Exit(1)
	case "sleep":
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	default:
		os.Exit(2)
	}
}

func helperRunner(t *testing.T, mode string, timeout time.Duration) Runner {
	t.Helper()
	t.Setenv("TEST_HELPER_PROCESS", "1")
	t.Setenv("TEST_GH_HELPER_MODE", mode)
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	base := newExecRunner(exe, timeout)
	return func(ctx context.Context, args ...string) ([]byte, error) {
		full := append([]string{"-test.run=TestHelperProcess", "--"}, args...)
		return base(ctx, full...)
	}
}

func TestExecRunner_Success(t *testing.T) {
	runner := helperRunner(t, "ok", DefaultCommandTimeout)
	out, err := runner(context.Background(), "issue", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != `[{"number":1}]` {
		t.Fatalf("unexpected stdout: %q", string(out))
	}
}

func TestExecRunner_RateLimit(t *testing.T) {
	runner := helperRunner(t, "rate_limit", DefaultCommandTimeout)
	_, err := runner(context.Background(), "issue", "list")
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestExecRunner_NonRateLimitError(t *testing.T) {
	runner := helperRunner(t, "error", DefaultCommandTimeout)
	_, err := runner(context.Background(), "auth", "status")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("did not expect ErrRateLimited, got %v", err)
	}
	if !strings.Contains(err.Error(), "not logged into any GitHub hosts") {
		t.Fatalf("expected stderr in error, got %v", err)
	}
}

func TestExecRunner_Timeout(t *testing.T) {
	runner := helperRunner(t, "sleep", 30*time.Millisecond)
	_, err := runner(context.Background(), "issue", "list")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
