package github

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kilip/opus/internal/tracker/domain"
)

// DefaultCommandTimeout is the per-command timeout applied to gh CLI invocations.
const DefaultCommandTimeout = 30 * time.Second

// Runner executes a gh CLI command and returns its stdout bytes.
type Runner func(ctx context.Context, args ...string) ([]byte, error)

// NewRunner returns a Runner that executes the `gh` CLI with a 30s per-command timeout.
func NewRunner() Runner {
	return newExecRunner("gh", DefaultCommandTimeout)
}

func newExecRunner(bin string, timeout time.Duration) Runner {
	return func(ctx context.Context, args ...string) ([]byte, error) {
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		cmd := exec.CommandContext(ctx, bin, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			errText := strings.TrimSpace(stderr.String())
			if strings.Contains(strings.ToLower(errText), "rate limit") {
				return nil, fmt.Errorf("%w: %s", domain.ErrRateLimited, errText)
			}
			if errText != "" {
				return nil, fmt.Errorf("%s %s: %w: %s", bin, strings.Join(args, " "), err, errText)
			}
			return nil, fmt.Errorf("%s %s: %w", bin, strings.Join(args, " "), err)
		}
		return stdout.Bytes(), nil
	}
}
