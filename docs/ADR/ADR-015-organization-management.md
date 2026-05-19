# ADR-015: Organization Management

**Status:** Accepted
**Date:** 2026-05-19
**Deciders:** Chief Architect, Product Manager
**Context:** Opus Server (`opus/server/`) · Opus Dash (`opus/dash/`)

---

## 1. Context

Opus is a self-hosted, autonomous AI assistant designed to operate on behalf of individuals
and teams. As Opus evolves toward multi-user support, it requires a formal organization
management architecture that governs how users are grouped, how membership is managed, and
how access is scoped across all feature domains.

Prior to this ADR, the concept of "workspace" existed informally across several ADRs —
primarily as the Casbin authorization domain in ADR-011 — without a formally defined domain
entity or lifecycle. This created ambiguity: there was no canonical definition of what a
workspace is, who owns it, or how users relate to it.

This ADR introduces the **Organization** as the top-level multi-user grouping entity in Opus,
inspired by the better-auth organization plugin model. It defines the organization domain,
membership lifecycle, role hierarchy, invitation system, personal organization convention, and
active organization session context. Workspace, as a child entity of Organization, is
intentionally out of scope and will be addressed in a dedicated ADR.

> **Note for AI agents and automated tooling:** This ADR is the authoritative specification
> for the Organization domain in Opus. Do not infer organization structure, role hierarchy,
> invitation flows, or API endpoints beyond what is defined here. The term "workspace" in
> earlier ADRs (particularly ADR-011) refers to what is now formally the Organization. ADR-011
> will be updated to reflect this change.

---

## 2. Decision

Opus Server adopts an **Organization-based multi-tenancy model** inspired by the better-auth
organization plugin. Every user belongs to at least one organization (their personal
organization, auto-created at registration). Users may create or join additional organizations.
All resource access is scoped to the active organization context carried in the session.

---

### 2.1 Core Principles

| Principle | Description |
|---|---|
| **Personal org by default** | Every user gets a personal organization auto-created at registration; no manual setup required |
| **Active org context** | Session always carries an `activeOrganizationId`; all API calls are scoped to it |
| **UUID v7 identifiers** | All entity IDs (Organization, Member, Invitation) use UUID v7 for time-ordered uniqueness |
| **Seamless solo UX** | Solo users never need to think about organizations; personal org is always active |
| **Multi-org capable** | Power users may create or join multiple organizations and switch between them |
| **Role hierarchy** | Three built-in roles: `owner`, `admin`, `member` — consistent with better-auth conventions |
| **Invitation system** | Members may be invited via email or shareable link; invitations are time-limited and configurable |
| **Slug-based identity** | Organizations have a unique human-readable slug for URL addressing |
| **Casbin domain migration** | Casbin authorization domain is now `organizationId`, replacing the informal `workspaceId` from ADR-011 |

---

### 2.2 Directory Structure

```
opus/
└── server/
    └── internal/
        ├── organization/
        │   ├── bootstrap.go        # Domain bootstrap: repo, service, handlers, events
        │   ├── model.go            # Organization, Member, Invitation, Role domain types
        │   ├── repository.go       # Repository interface (port) + //go:generate directive
        │   ├── service.go          # Business logic
        │   ├── config.go           # organization.Config (hybrid composition — ADR-002)
        │   └── errors.go           # Sentinel errors
        │
        └── adapter/
            └── entgo/
                └── organization.go # Implements organization.Repository
```

---

### 2.3 Domain Models

```go
// internal/organization/model.go
package organization

import "time"

// Organization represents a group of users collaborating within Opus.
// Every user has exactly one personal Organization (IsPersonal: true),
// created automatically at registration.
type Organization struct {
    ID          string     // UUID v7
    Name        string
    Slug        string     // Unique human-readable identifier (e.g. "acme-corp")
    LogoURL     string
    IsPersonal  bool       // True for auto-created personal organizations
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Member represents the relationship between a User and an Organization.
// A user may belong to multiple organizations; each membership carries a Role.
type Member struct {
    ID             string    // UUID v7
    OrganizationID string    // UUID v7
    UserID         string    // UUID v7
    Role           Role
    CreatedAt      time.Time
}

// Invitation represents a pending request for a user to join an Organization.
// Invitations may be issued to a specific email address or as a shareable link.
type Invitation struct {
    ID             string    // UUID v7
    OrganizationID string    // UUID v7
    Email          string     // Empty for link-based invitations
    Role           Role       // Role assigned upon acceptance
    InviterID      string     // UserID of the member who created the invitation (UUID v7)
    Token          string     // Opaque token used to accept the invitation
    Type           InvitationType
    Status         InvitationStatus
    ExpiresAt      time.Time
    CreatedAt      time.Time
}

// Role defines a member's permissions within an organization.
type Role string

const (
    // RoleOwner has full control over the organization, including deletion
    // and ownership transfer. The last owner cannot be removed.
    RoleOwner Role = "owner"

    // RoleAdmin may manage members and update organization settings,
    // but cannot delete the organization or change the owner.
    RoleAdmin Role = "admin"

    // RoleMember has read access to organization resources.
    // Write access to specific resources is governed by Casbin policy.
    RoleMember Role = "member"
)

// InvitationType distinguishes between targeted email invitations
// and shareable link-based invitations.
type InvitationType string

const (
    // InvitationTypeEmail targets a specific email address.
    InvitationTypeEmail InvitationType = "email"

    // InvitationTypeLink is a shareable token that any user may redeem.
    InvitationTypeLink InvitationType = "link"
)

// InvitationStatus represents the lifecycle state of an Invitation.
type InvitationStatus string

const (
    // InvitationStatusPending is the initial state of a newly created invitation.
    InvitationStatusPending InvitationStatus = "pending"

    // InvitationStatusAccepted indicates the invitation has been redeemed.
    InvitationStatusAccepted InvitationStatus = "accepted"

    // InvitationStatusRejected indicates the invitee explicitly declined.
    InvitationStatusRejected InvitationStatus = "rejected"

    // InvitationStatusCancelled indicates the invitation was revoked by an admin or owner.
    InvitationStatusCancelled InvitationStatus = "cancelled"

    // InvitationStatusExpired indicates the invitation passed its ExpiresAt without being redeemed.
    InvitationStatusExpired InvitationStatus = "expired"
)
```

---

### 2.4 Repository Interface

```go
// internal/organization/repository.go
package organization

import "context"

//go:generate mockgen -destination=mock_repository.go -package=organization . Repository

// Repository defines the persistence contract for the Organization domain.
type Repository interface {
    // Organization CRUD
    FindOrganizationByID(ctx context.Context, id string) (*Organization, error)
    FindOrganizationBySlug(ctx context.Context, slug string) (*Organization, error)
    FindOrganizationsByUserID(ctx context.Context, userID string) ([]*Organization, error)
    CreateOrganization(ctx context.Context, org *Organization) (*Organization, error)
    UpdateOrganization(ctx context.Context, org *Organization) (*Organization, error)
    DeleteOrganization(ctx context.Context, id string) error

    // Member management
    FindMemberByID(ctx context.Context, id string) (*Member, error)
    FindMemberByUserAndOrg(ctx context.Context, userID, organizationID string) (*Member, error)
    FindMembersByOrgID(ctx context.Context, organizationID string, cursor string, limit int) ([]*Member, string, error)
    CreateMember(ctx context.Context, member *Member) (*Member, error)
    UpdateMemberRole(ctx context.Context, memberID string, role Role) error
    DeleteMember(ctx context.Context, memberID string) error
    CountOwnersByOrgID(ctx context.Context, organizationID string) (int, error)

    // Invitation management
    FindInvitationByID(ctx context.Context, id string) (*Invitation, error)
    FindInvitationByToken(ctx context.Context, token string) (*Invitation, error)
    FindInvitationsByOrgID(ctx context.Context, organizationID string) ([]*Invitation, error)
    CreateInvitation(ctx context.Context, invitation *Invitation) (*Invitation, error)
    UpdateInvitationStatus(ctx context.Context, id string, status InvitationStatus) error
}
```

---

### 2.5 Service Layer

```go
// internal/organization/service.go
package organization

import (
    "context"
    "fmt"
    "time"

    "github.com/kilip/opus/server/internal/shared/logger"
    "github.com/kilip/opus/server/internal/shared/queue"
)

// Service handles all business logic for the Organization domain.
type Service struct {
    repo   Repository
    bus    queue.EventBus
    logger logger.Logger
    cfg    Config
}

// NewService constructs a new Service with the provided dependencies.
func NewService(repo Repository, bus queue.EventBus, log logger.Logger, cfg Config) *Service {
    return &Service{
        repo:   repo,
        bus:    bus,
        logger: log.With(logger.String("component", "organization_service")),
        cfg:    cfg,
    }
}

// CreatePersonalOrganization creates a personal organization for a newly registered user.
// The user is automatically assigned the owner role.
// Called by the auth domain after successful user registration.
func (s *Service) CreatePersonalOrganization(ctx context.Context, userID, userName string) (*Organization, error) {
    slug := slugify(userName)
    org := &Organization{
        Name:       userName,
        Slug:       slug,
        IsPersonal: true,
    }
    created, err := s.repo.CreateOrganization(ctx, org)
    if err != nil {
        return nil, fmt.Errorf("organization.Service.CreatePersonalOrganization: %w", err)
    }

    _, err = s.repo.CreateMember(ctx, &Member{
        OrganizationID: created.ID,
        UserID:         userID,
        Role:           RoleOwner,
    })
    if err != nil {
        return nil, fmt.Errorf("organization.Service.CreatePersonalOrganization: %w", err)
    }

    if err := s.bus.Publish(ctx, queue.Event{
        Topic:   "organization.created",
        Payload: mustMarshal(created),
        Source:  "organization",
    }); err != nil {
        s.logger.WarnCtx(ctx, "failed to publish organization.created event",
            logger.Err(err),
        )
    }

    return created, nil
}

// CreateOrganization creates a new non-personal organization.
// The creator is automatically assigned the owner role.
func (s *Service) CreateOrganization(ctx context.Context, userID, name, slug string) (*Organization, error) {
    existing, _ := s.repo.FindOrganizationBySlug(ctx, slug)
    if existing != nil {
        return nil, fmt.Errorf("organization.Service.CreateOrganization: %w", ErrSlugTaken)
    }

    org := &Organization{
        Name:       name,
        Slug:       slug,
        IsPersonal: false,
    }
    created, err := s.repo.CreateOrganization(ctx, org)
    if err != nil {
        return nil, fmt.Errorf("organization.Service.CreateOrganization: %w", err)
    }

    _, err = s.repo.CreateMember(ctx, &Member{
        OrganizationID: created.ID,
        UserID:         userID,
        Role:           RoleOwner,
    })
    if err != nil {
        return nil, fmt.Errorf("organization.Service.CreateOrganization: %w", err)
    }

    return created, nil
}

// InviteMember creates an invitation for a user to join an organization.
// The caller must be an owner or admin of the organization.
func (s *Service) InviteMember(ctx context.Context, organizationID, inviterID, email string, role Role, invType InvitationType) (*Invitation, error) {
    inviter, err := s.repo.FindMemberByUserAndOrg(ctx, inviterID, organizationID)
    if err != nil {
        return nil, fmt.Errorf("organization.Service.InviteMember: %w", ErrNotMember)
    }
    if inviter.Role == RoleMember {
        return nil, fmt.Errorf("organization.Service.InviteMember: %w", ErrInsufficientRole)
    }

    ttl := s.cfg.InvitationTTL
    inv := &Invitation{
        OrganizationID: organizationID,
        Email:          email,
        Role:           role,
        InviterID:      inviterID,
        Token:          generateToken(),
        Type:           invType,
        Status:         InvitationStatusPending,
        ExpiresAt:      time.Now().Add(ttl),
    }

    created, err := s.repo.CreateInvitation(ctx, inv)
    if err != nil {
        return nil, fmt.Errorf("organization.Service.InviteMember: %w", err)
    }

    if err := s.bus.Publish(ctx, queue.Event{
        Topic:   "organization.invitation_created",
        Payload: mustMarshal(created),
        Source:  "organization",
    }); err != nil {
        s.logger.WarnCtx(ctx, "failed to publish organization.invitation_created event",
            logger.Err(err),
        )
    }

    return created, nil
}

// AcceptInvitation redeems an invitation token and adds the user as a member.
func (s *Service) AcceptInvitation(ctx context.Context, token, userID string) (*Member, error) {
    inv, err := s.repo.FindInvitationByToken(ctx, token)
    if err != nil {
        return nil, fmt.Errorf("organization.Service.AcceptInvitation: %w", ErrInvitationNotFound)
    }
    if inv.Status != InvitationStatusPending {
        return nil, fmt.Errorf("organization.Service.AcceptInvitation: %w", ErrInvitationNotPending)
    }
    if time.Now().After(inv.ExpiresAt) {
        _ = s.repo.UpdateInvitationStatus(ctx, inv.ID, InvitationStatusExpired)
        return nil, fmt.Errorf("organization.Service.AcceptInvitation: %w", ErrInvitationExpired)
    }

    member, err := s.repo.CreateMember(ctx, &Member{
        OrganizationID: inv.OrganizationID,
        UserID:         userID,
        Role:           inv.Role,
    })
    if err != nil {
        return nil, fmt.Errorf("organization.Service.AcceptInvitation: %w", err)
    }

    if err := s.repo.UpdateInvitationStatus(ctx, inv.ID, InvitationStatusAccepted); err != nil {
        return nil, fmt.Errorf("organization.Service.AcceptInvitation: %w", err)
    }

    return member, nil
}

// RemoveMember removes a member from an organization.
// The last owner of an organization cannot be removed.
func (s *Service) RemoveMember(ctx context.Context, organizationID, memberID, callerID string) error {
    member, err := s.repo.FindMemberByID(ctx, memberID)
    if err != nil {
        return fmt.Errorf("organization.Service.RemoveMember: %w", ErrNotMember)
    }

    if member.Role == RoleOwner {
        count, err := s.repo.CountOwnersByOrgID(ctx, organizationID)
        if err != nil {
            return fmt.Errorf("organization.Service.RemoveMember: %w", err)
        }
        if count <= 1 {
            return fmt.Errorf("organization.Service.RemoveMember: %w", ErrLastOwner)
        }
    }

    if err := s.repo.DeleteMember(ctx, memberID); err != nil {
        return fmt.Errorf("organization.Service.RemoveMember: %w", err)
    }

    return nil
}

// UpdateMemberRole updates a member's role within an organization.
// The last owner's role cannot be changed away from owner.
func (s *Service) UpdateMemberRole(ctx context.Context, organizationID, memberID string, newRole Role) error {
    member, err := s.repo.FindMemberByID(ctx, memberID)
    if err != nil {
        return fmt.Errorf("organization.Service.UpdateMemberRole: %w", ErrNotMember)
    }

    if member.Role == RoleOwner && newRole != RoleOwner {
        count, err := s.repo.CountOwnersByOrgID(ctx, organizationID)
        if err != nil {
            return fmt.Errorf("organization.Service.UpdateMemberRole: %w", err)
        }
        if count <= 1 {
            return fmt.Errorf("organization.Service.UpdateMemberRole: %w", ErrLastOwner)
        }
    }

    if err := s.repo.UpdateMemberRole(ctx, memberID, newRole); err != nil {
        return fmt.Errorf("organization.Service.UpdateMemberRole: %w", err)
    }

    return nil
}
```

---

### 2.6 Sentinel Errors

```go
// internal/organization/errors.go
package organization

import "errors"

// ErrNotFound is returned when a requested organization does not exist.
var ErrNotFound = errors.New("organization: not found")

// ErrSlugTaken is returned when a requested organization slug is already in use.
var ErrSlugTaken = errors.New("organization: slug already taken")

// ErrNotMember is returned when a user is not a member of the requested organization.
var ErrNotMember = errors.New("organization: user is not a member")

// ErrInsufficientRole is returned when a member's role does not permit the requested action.
var ErrInsufficientRole = errors.New("organization: insufficient role for this action")

// ErrLastOwner is returned when an operation would remove or demote the last owner
// of an organization, which is prohibited.
var ErrLastOwner = errors.New("organization: cannot remove or demote the last owner")

// ErrPersonalOrgImmutable is returned when an attempt is made to delete or rename
// a personal organization.
var ErrPersonalOrgImmutable = errors.New("organization: personal organization cannot be deleted or renamed")

// ErrInvitationNotFound is returned when an invitation token does not exist.
var ErrInvitationNotFound = errors.New("organization: invitation not found")

// ErrInvitationNotPending is returned when an invitation has already been acted upon.
var ErrInvitationNotPending = errors.New("organization: invitation is no longer pending")

// ErrInvitationExpired is returned when an invitation token has passed its expiry time.
var ErrInvitationExpired = errors.New("organization: invitation has expired")
```

---

### 2.7 Configuration — Hybrid Composition (ADR-002)

```go
// internal/organization/config.go
package organization

import "time"

// Config holds all organization domain configuration.
// Owned by the organization package; composed into the root config.Config
// by internal/config/model.go.
//
// Environment variable overrides:
//   OPUS_ORGANIZATION_INVITATION_TTL            — sets InvitationTTL
//   OPUS_ORGANIZATION_ALLOW_USER_CREATE         — sets AllowUserCreate
type Config struct {
    // InvitationTTL is the duration for which an invitation token remains valid.
    // Default: 48h. Configurable to support stricter or more permissive policies.
    InvitationTTL time.Duration `mapstructure:"invitation_ttl" json:"invitation_ttl" jsonschema:"default=48h,description=Duration for which an invitation token remains valid (Go duration string)"`

    // AllowUserCreate controls whether any authenticated user may create a new
    // organization. Default: true. Set to false to restrict org creation to
    // system administrators only.
    AllowUserCreate bool `mapstructure:"allow_user_create" json:"allow_user_create" jsonschema:"default=true,description=Allow any authenticated user to create a new organization"`
}
```

Root config composition in `internal/config/model.go`:

```go
// internal/config/model.go (excerpt)
import "github.com/kilip/opus/server/internal/organization"

type Config struct {
    // ... existing fields ...
    Organization organization.Config `mapstructure:"organization" json:"organization"`
}
```

---

### 2.8 Ent Schema

```go
// ent/schema/organization.go
package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

// Organization holds the schema definition for the Organization entity.
type Organization struct{ ent.Schema }

func (Organization) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").Unique().Immutable(), // UUID v7
        field.String("name").NotEmpty(),
        field.String("slug").Unique().NotEmpty(),
        field.String("logo_url").Optional(),
        field.Bool("is_personal").Default(false),
        field.Time("created_at").Immutable(),
        field.Time("updated_at"),
    }
}

func (Organization) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("slug"),
    }
}
```

```go
// ent/schema/org_member.go
package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

// OrgMember holds the schema definition for the OrgMember entity.
type OrgMember struct{ ent.Schema }

func (OrgMember) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").Unique().Immutable(), // UUID v7
        field.String("organization_id").NotEmpty(),
        field.String("user_id").NotEmpty(),
        field.Enum("role").Values("owner", "admin", "member").Default("member"),
        field.Time("created_at").Immutable(),
    }
}

func (OrgMember) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("organization_id", "user_id").Unique(),
        index.Fields("user_id"),
    }
}
```

```go
// ent/schema/org_invitation.go
package schema

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

// OrgInvitation holds the schema definition for the OrgInvitation entity.
type OrgInvitation struct{ ent.Schema }

func (OrgInvitation) Fields() []ent.Field {
    return []ent.Field{
        field.String("id").Unique().Immutable(), // UUID v7
        field.String("organization_id").NotEmpty(),
        field.String("email").Optional(),
        field.Enum("role").Values("owner", "admin", "member").Default("member"),
        field.String("inviter_id").NotEmpty(),
        field.String("token").Unique().NotEmpty(),
        field.Enum("type").Values("email", "link").Default("email"),
        field.Enum("status").Values("pending", "accepted", "rejected", "cancelled", "expired").Default("pending"),
        field.Time("expires_at"),
        field.Time("created_at").Immutable(),
    }
}

func (OrgInvitation) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("token"),
        index.Fields("organization_id", "status"),
        index.Fields("email", "organization_id"),
    }
}
```

---

### 2.9 Active Organization Session Context

The active organization is tracked in the session. The `activeOrganizationId` field is stored
in the `sessions` table and propagated through JWT claims on every authenticated request.

#### 2.9.1 JWT Claims Update (ADR-011 Amendment)

The `Claims` struct in `internal/auth/model.go` is updated to replace the informal
`WorkspaceID` field with `ActiveOrganizationID`:

```go
// internal/auth/model.go (updated Claims)
type Claims struct {
    Sub                    string    `json:"sub"`
    SessionID              string    `json:"sid"`
    ActiveOrganizationID   string    `json:"aoid"`   // replaces wid
    Role                   string    `json:"role"`
    IssuedAt               time.Time `json:"iat"`
    ExpiresAt              time.Time `json:"exp"`
}
```

#### 2.9.2 Switch Active Organization

Users switch the active organization via a dedicated endpoint. The server validates
membership, updates the session record, and issues a new token pair with the updated
`ActiveOrganizationID`.

```
POST /auth/organizations/switch
Body: { "organizationId": "org_01ABCDEF" }
```

**Rules:**
- The user must be an active member of the target organization.
- Personal organizations are always switchable regardless of other memberships.
- A successful switch issues a new access token and refreshes the session.

#### 2.9.3 Personal Organization Auto-Activation

When a user registers:

1. Auth domain creates the user record.
2. Auth domain publishes `auth.user_registered` event via EventBus.
3. Organization domain subscribes to `auth.user_registered` and calls
   `CreatePersonalOrganization`.
4. Auth domain subscribes to `organization.created` (where `IsPersonal: true`) and sets
   `activeOrganizationId` in the new session.

This ensures the user's first session always has a valid `activeOrganizationId` without
any manual step.

---

### 2.10 Casbin Domain Migration (ADR-011 Amendment)

The Casbin authorization domain is updated from the informal `workspaceId` to the formal
`organizationId`. All existing Casbin policy records must be migrated as part of the
database migration for this ADR.

**Updated Casbin policy format:**

```
(userID, organizationID, resource, action)
```

**Example policies:**

```
p, owner, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, agent,    read
p, owner, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, agent,    write
p, owner, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, agent,    delete
p, owner, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, agent,    manage
p, owner, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, vault,    read
p, owner, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, vault,    write
p, owner, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, vault,    delete
p, owner, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, workflow, read
p, owner, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, workflow, write
p, owner, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, workflow, delete
p, owner, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, user,     manage

p, admin, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, agent,    read
p, admin, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, agent,    write
p, admin, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, vault,    read
p, admin, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, vault,    write
p, admin, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, workflow, read
p, admin, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, workflow, write

p, member, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, agent,    read
p, member, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, vault,    read
p, member, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90, workflow, read
```

**Role assignment:**

```
g, 018f3a5a-3c2b-7d1e-8f9g-0h1i2j3k4l5m, owner,  018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90
g, 018f3a5b-4d5e-6f7g-8h9i-0j1k2l3m4n5o, admin,  018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90
g, 018f3a5c-5f6g-7h8i-9j0k-1l2m3n4o5p6q, member, 018f3a5a-8b3d-7a2e-9f1c-4b5c6d7e8f90
```

Casbin policies are seeded automatically when an organization is created, based on the
built-in role definitions. The `PolicyService` in `internal/auth/casbin.go` is responsible
for seeding and revoking policies on membership lifecycle events.

---

### 2.11 API Endpoints

All endpoints follow the ADR-004 response envelope convention.

| Method | Path | Role Required | Description |
|---|---|---|---|
| `POST` | `/organizations` | Authenticated | Create a new organization |
| `GET` | `/organizations` | Authenticated | List organizations the caller belongs to |
| `GET` | `/organizations/:slug` | Member | Get organization by slug |
| `PATCH` | `/organizations/:id` | Admin, Owner | Update organization name or logo |
| `DELETE` | `/organizations/:id` | Owner | Delete organization (non-personal only) |
| `GET` | `/organizations/:id/members` | Member | List members (cursor-paginated) |
| `PATCH` | `/organizations/:id/members/:memberId` | Admin, Owner | Update member role |
| `DELETE` | `/organizations/:id/members/:memberId` | Admin, Owner | Remove member |
| `POST` | `/organizations/:id/invitations` | Admin, Owner | Create invitation (email or link) |
| `GET` | `/organizations/:id/invitations` | Admin, Owner | List invitations |
| `DELETE` | `/organizations/:id/invitations/:invitationId` | Admin, Owner | Cancel invitation |
| `POST` | `/invitations/:token/accept` | Authenticated | Accept an invitation by token |
| `POST` | `/invitations/:token/reject` | Authenticated | Reject an invitation by token |
| `POST` | `/auth/organizations/switch` | Authenticated | Switch active organization |

**Personal organization constraints enforced at handler level:**

- `DELETE /organizations/:id` returns `403 Forbidden` if the organization has `isPersonal: true`.
- `PATCH /organizations/:id` returns `403 Forbidden` for name changes on personal organizations.

---

### 2.12 Domain Events

The organization domain publishes the following events via `queue.EventBus`. All event topics
follow the `"<domain>.<action>"` convention established in ADR-008.

| Topic | Payload | Consumers |
|---|---|---|
| `organization.created` | `{ id, name, slug, isPersonal }` | `auth` (set activeOrganizationId on personal org) |
| `organization.deleted` | `{ id }` | `vault`, `agent`, `workflow` (cleanup scoped resources) |
| `organization.member_added` | `{ organizationId, userId, role }` | `auth` (seed Casbin policy) |
| `organization.member_removed` | `{ organizationId, userId }` | `auth` (revoke Casbin policy) |
| `organization.member_role_updated` | `{ organizationId, userId, oldRole, newRole }` | `auth` (update Casbin policy) |
| `organization.invitation_created` | `{ id, organizationId, email, token, type }` | `gmail` (send invitation email if type=email) |

The organization domain subscribes to:

| Topic | Producer | Handler |
|---|---|---|
| `auth.user_registered` | `auth` | `OnUserRegistered` — creates personal organization |

---

### 2.13 Bootstrap

```go
// internal/organization/bootstrap.go
package organization

import (
    "github.com/kilip/opus/server/ent"
    "github.com/kilip/opus/server/internal/adapter/entgo"
    "github.com/kilip/opus/server/internal/shared/logger"
    "github.com/kilip/opus/server/internal/shared/queue"
)

// Bootstrap initialises the organization domain: repository, service,
// job handlers, and event subscriptions.
// Called by container.Bootstrap() during startup.
func Bootstrap(
    db  *ent.Client,
    bus queue.EventBus,
    q   queue.Queue,
    log logger.Logger,
    cfg Config,
) {
    repo := entgo.NewOrganizationRepo(db)
    svc  := NewService(repo, bus, log, cfg)

    // Subscribe to domain events from other domains.
    bus.Subscribe("auth.user_registered", svc.OnUserRegistered)

    setService(svc)
}

var svc *Service

func setService(s *Service) { svc = s }

// GetService returns the initialised organization.Service.
// Panics if Bootstrap has not been called.
func GetService() *Service {
    if svc == nil {
        panic("organization: Bootstrap has not been called")
    }
    return svc
}
```

`container/bootstrap.go` addition — organization must be bootstrapped **after** auth:

```go
// internal/container/bootstrap.go (excerpt)
func Bootstrap(cfg *config.Config) {
    initShared(cfg)
    auth.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.Auth)
    organization.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.Organization) // ← after auth
    vault.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.Vault)
    agent.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.Agent)
    workflow.Bootstrap(c.db, c.bus, c.queue, c.log, cfg.Workflow)
    // ... other domains ...
    fiberdelivery.Bootstrap(c.fiber, c.log, cfg.Server)
}
```

---

### 2.14 Dash Integration

Opus Dash exposes organization management under the `features/organization/` module,
consistent with the feature-based architecture defined in ADR-003.

```
dash/src/
└── features/
    └── organization/
        ├── components/
        │   ├── OrgSwitcher.tsx        # Active org switcher in sidebar
        │   ├── OrgSettingsForm.tsx    # Update org name and logo
        │   ├── MemberList.tsx         # Paginated member table
        │   ├── InviteMemberModal.tsx  # Email or link invitation form
        │   └── InvitationList.tsx     # Pending invitations table
        ├── hooks/
        │   ├── useActiveOrganization.ts
        │   ├── useOrganizationMembers.ts
        │   └── useOrganizationInvitations.ts
        ├── api.ts                     # TanStack Query queryKeys + queryFn
        └── types.ts                   # Organization, Member, Invitation TS types
```

**OrgSwitcher behaviour:**

- Always visible in the sidebar.
- Personal organization is listed first, labelled with the user's name.
- Selecting an organization calls `POST /auth/organizations/switch`, invalidates all
  TanStack Query caches, and redirects to `/agent`.
- Solo users (only personal org) see the switcher but it has no actionable items — this
  is intentional for future discoverability.

---

### 2.15 Example `config.json` Addition

```json
{
  "organization": {
    "invitation_ttl": "48h",
    "allow_user_create": true
  }
}
```

---

## 3. Alternatives Considered

### 3.1 Retain Informal Workspace as Top-Level Entity

Keep the existing `workspaceId` concept from ADR-011 without introducing a formal
Organization domain. Rejected because:

- `workspaceId` was never formally defined as a domain entity — it existed only as a
  Casbin domain string with no associated lifecycle, membership model, or API.
- Multi-user support requires a proper membership model (roles, invitations, ownership)
  that cannot be retrofitted onto an informal string identifier.
- Formalising as Organization now prevents a more disruptive migration later.

### 3.2 Flat Role Model (No Owner/Admin/Member Hierarchy)

A single `admin` / `user` role model without an `owner` concept. Rejected because:

- Without an owner role, there is no protection against the last administrator being
  removed, which could leave an organization in an unmanageable state.
- The owner / admin / member hierarchy is a well-understood convention (GitHub, Slack,
  Linear, better-auth) that reduces onboarding friction.

### 3.3 Merge Organization and Workspace

Treat Organization and Workspace as the same entity (as Notion does with its top-level
Workspace concept). Rejected because:

- Opus has a more complex resource hierarchy than Notion — agents, vaults, and workflows
  benefit from sub-organization scoping (Workspace) that is distinct from the top-level
  membership boundary (Organization).
- Keeping Organization and Workspace separate allows each to evolve independently.
- A future Workspace ADR can define the parent-child relationship without modifying the
  Organization domain.

### 3.4 Email-Only Invitations

Support only email-based invitations, not shareable links. Rejected because:

- Shareable links are essential for self-hosted deployments where outbound email may not
  be configured.
- Many self-hosted users will be inviting known colleagues; a link shared via Slack or
  a messaging app is more practical than configuring SMTP.

---

## 4. Consequences

### 4.1 Positive

- **Formal multi-tenancy** — Organization provides a well-defined boundary for all
  resource scoping, replacing the informal `workspaceId` string.
- **Seamless solo UX** — Personal organizations are transparent to solo users; no
  onboarding friction is introduced for the primary Opus use case.
- **Extensible role model** — The owner / admin / member hierarchy covers MVP and
  leaves room for workspace-level roles in a future ADR.
- **Invitation flexibility** — Both email and link invitations support self-hosted
  deployments with and without SMTP configuration.
- **EventBus decoupling** — Organization lifecycle events (member added, removed,
  role updated) drive Casbin policy seeding without direct domain coupling.
- **better-auth alignment** — Consistent naming and behavior with better-auth reduces
  confusion for contributors familiar with that ecosystem.
- **Clean ADR-012 addition** — Adding the organization domain requires only four steps
  per the process defined in ADR-012.

### 4.2 Negative / Trade-offs

- **ADR-011 amendment required** — JWT `Claims` must be updated (`wid` → `aoid`) and
  existing sessions invalidated on deployment. This is a one-time migration cost.
- **Casbin policy migration** — Existing Casbin policy records using `workspaceId` as
  domain must be migrated to `organizationId`. An Atlas migration handles this
  automatically.
- **Personal org complexity** — The `isPersonal` flag adds a special case to several
  service methods and handler guards. This is intentional but must be consistently
  enforced.
- **Switch endpoint latency** — Switching the active organization requires a token
  reissue, which involves a database write and a new JWT signing operation. Acceptable
  for an infrequent operation; not suitable for per-request org switching.
- **EventBus ordering dependency** — `organization.Bootstrap` must be called after
  `auth.Bootstrap` in `container/bootstrap.go` because organization subscribes to
  `auth.user_registered`. Incorrect ordering causes a startup-time panic.

---

## 5. References

- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)
- [ADR-002: Configuration Management](./ADR-002-server-configuration.md)
- [ADR-003: Opus Dash Frontend Architecture](./ADR-003-dash-frontend-architecture.md)
- [ADR-004: API Response Contract](./ADR-004-api-response-contract.md)
- [ADR-007: ORM and Database Strategy](./ADR-007-orm-and-database-strategy.md)
- [ADR-008: Server Queue Architecture](./ADR-008-server-queue.md)
- [ADR-010: Server Coding Conventions & Linting](./ADR-010-server-coding-and-linting.md)
- [ADR-011: Authentication and Authorization Architecture](./ADR-011-authentication-and-authorization.md)
- [ADR-012: Module System and Dependency Injection](./ADR-012-module-system-and-dependency-injection.md)
- [better-auth Organization Plugin](https://better-auth.com/docs/plugins/organization)
- [better-auth Organization — Members, Roles & Invitations](https://deepwiki.com/better-auth/better-auth/5.2-organization-plugin)
- [Casbin Domain-Based RBAC](https://casbin.org/docs/rbac-with-domains)
