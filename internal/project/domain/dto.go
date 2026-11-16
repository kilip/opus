package domain

import "time"

// Project represents the project entity DTO.
type Project struct {
	ID              string    `json:"id"`
	Slug            string    `json:"slug"`
	Name            string    `json:"name"`
	RepoProvider    string    `json:"repo_provider"`
	RepoRef         string    `json:"repo_ref"`
	TrackerProvider string    `json:"tracker_provider"`
	TrackerRef      string    `json:"tracker_ref"`
	DefaultBranch   string    `json:"default_branch"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateProjectRequest is the payload for creating a project.
type CreateProjectRequest struct {
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	RepoProvider    string `json:"repo_provider"`
	RepoRef         string `json:"repo_ref"`
	TrackerProvider string `json:"tracker_provider,omitempty"`
	TrackerRef      string `json:"tracker_ref,omitempty"`
	DefaultBranch   string `json:"default_branch,omitempty"`
}

// UpdateProjectRequest is the payload for updating a project.
type UpdateProjectRequest struct {
	Name            *string `json:"name,omitempty"`
	TrackerProvider *string `json:"tracker_provider,omitempty"`
	TrackerRef      *string `json:"tracker_ref,omitempty"`
	DefaultBranch   *string `json:"default_branch,omitempty"`
}
