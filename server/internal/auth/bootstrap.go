package auth

import (
	"fmt"
	"github.com/casbin/casbin/v2"
	"github.com/kilip/opus/server/internal/shared/logger"
	"github.com/kilip/opus/server/internal/shared/queue"
	"os"
)

var (
	svc  *Service
	repo Repository
)

// Bootstrap initializes the auth domain.
func Bootstrap(r Repository, bus queue.EventBus, q queue.Queue, log logger.Logger, cfg Config) {
	repo = r

	// Initialize Casbin for PolicyService
	// Try different paths to find casbin_model.conf
	paths := []string{
		"internal/auth/casbin_model.conf",
		"casbin_model.conf",
		"server/internal/auth/casbin_model.conf",
		"../auth/casbin_model.conf",
		"../../internal/auth/casbin_model.conf",
	}

	var enforcer *casbin.Enforcer
	var err error
	for _, p := range paths {
		if _, e := os.Stat(p); e == nil {
			enforcer, err = casbin.NewEnforcer(p)
			if err == nil {
				break
			}
		}
	}

	if enforcer == nil {
		panic(fmt.Errorf("auth: failed to create casbin enforcer: %w", err))
	}
	policySvc := NewCasbinPolicyManager(enforcer)

	providerRegistry := NewProviderRegistry()

	s := NewService(repo, providerRegistry, policySvc, cfg, log)
	setService(s)
}

func setService(s *Service) {
	svc = s
}

// GetService returns the initialized auth service.
// Panics if Bootstrap has not been called.
func GetService() *Service {
	if svc == nil {
		panic("auth: service not initialized. call Bootstrap() first")
	}
	return svc
}

// GetRepository returns the initialized auth repository.
// Panics if Bootstrap has not been called.
func GetRepository() Repository {
	if repo == nil {
		panic("auth: repository not initialized. call Bootstrap() first")
	}
	return repo
}
