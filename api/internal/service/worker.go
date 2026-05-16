package service

import (
	"context"

	"github.com/kilip/opus/api/internal/model"
)

// HandlerFunc is the function signature all job handlers must implement.
type HandlerFunc func(ctx context.Context, job *model.Job) error

// Worker defines the worker engine lifecycle.
type Worker interface {
	// Register associates a job type string with a handler function.
	Register(jobType string, handler HandlerFunc)
	// Start launches N worker goroutines and begins consuming jobs.
	Start(ctx context.Context) error
	// Stop signals all workers to finish in-flight jobs and shut down.
	Stop() error
}
