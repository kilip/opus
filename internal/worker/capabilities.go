package worker

import (
	"context"
	"os/exec"

	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/worker/domain"
	"github.com/kilip/opus/internal/worker/provider/agy"
)

// CapabilityProbe detects local tool availability and authentication state.
type CapabilityProbe func(ctx context.Context) domain.Capabilities

// DefaultCapabilityProbe checks PATH availability and auth status for agy and gh.
func DefaultCapabilityProbe(ctx context.Context) domain.Capabilities {
	return NewCapabilityProbe("")(ctx)
}

// NewCapabilityProbe returns a CapabilityProbe using agyBin (defaults to WORKER_AGY_BIN / "agy").
func NewCapabilityProbe(agyBin string) CapabilityProbe {
	return func(ctx context.Context) domain.Capabilities {
		return domain.Capabilities{
			Agy: probeAgy(ctx, agyBin),
			Gh:  probeGh(),
		}
	}
}

func probeAgy(ctx context.Context, agyBin string) domain.ToolStatus {
	installed, authenticated := agy.ProbeAuth(ctx, agyBin)
	if !installed {
		return node.ToolStatusMissing
	}
	if !authenticated {
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
