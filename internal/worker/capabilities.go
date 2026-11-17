package worker

import (
	"context"
	"os/exec"
	"time"

	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/worker/domain"
)

const capabilityProbeTimeout = 3 * time.Second

// CapabilityProbe detects local tool availability and authentication state.
type CapabilityProbe func(ctx context.Context) domain.Capabilities

// DefaultCapabilityProbe checks PATH availability and auth status for agy and gh.
func DefaultCapabilityProbe(ctx context.Context) domain.Capabilities {
	return domain.Capabilities{
		Agy: probeAgy(ctx),
		Gh:  probeGh(),
	}
}

func probeAgy(ctx context.Context) domain.ToolStatus {
	if _, err := exec.LookPath("agy"); err != nil {
		return node.ToolStatusMissing
	}
	probeCtx, cancel := context.WithTimeout(ctx, capabilityProbeTimeout)
	defer cancel()
	if err := exec.CommandContext(probeCtx, "agy", "auth", "status").Run(); err != nil {
		return node.ToolStatusUnauthenticated
	}
	return node.ToolStatusReady
}

func probeGh() domain.ToolStatus {
	if _, err := exec.LookPath("gh"); err != nil {
		return node.ToolStatusMissing
	}
	return node.ToolStatusReady
}
