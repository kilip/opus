package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kilip/opus/internal/task/domain"
	"github.com/kilip/opus/internal/task/repo"
	"github.com/kilip/opus/internal/task/service"
)

func newService() *service.Task {
	return service.New(repo.NewMemory(func() time.Time {
		return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	}))
}

func TestCreateRejectsBlankTitle(t *testing.T) {
	_, err := newService().Create(domain.CreateTaskRequest{Title: "  "})
	if !errors.Is(err, domain.ErrTaskTitleEmpty) {
		t.Fatalf("expected ErrTaskTitleEmpty, got %v", err)
	}
}

func TestUpdateRejectsBlankTitle(t *testing.T) {
	blank := ""
	_, err := newService().Update("t1", domain.UpdateTaskRequest{Title: &blank})
	if !errors.Is(err, domain.ErrTaskTitleEmpty) {
		t.Fatalf("expected ErrTaskTitleEmpty, got %v", err)
	}
}

func TestCreateActivityValidatesBeforeLookup(t *testing.T) {
	s := newService()

	_, err := s.CreateActivity("t1", domain.CreateActivityRequest{Body: " "})
	if !errors.Is(err, domain.ErrActivityBodyEmpty) {
		t.Fatalf("expected ErrActivityBodyEmpty, got %v", err)
	}

	_, err = s.CreateActivity("missing", domain.CreateActivityRequest{Body: "hi"})
	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}
