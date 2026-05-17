# ADR-004: API Response Contract

**Status:** Accepted
**Date:** 2026-05-17
**Deciders:** Chief Architect
**Context:** Opus Server (`opus/server/`) · Opus Dash (`opus/dash/`)

---

## 1. Context

Opus Server exposes a REST + SSE API consumed by Opus Dash and any future clients (mobile applications, CLI tools, third-party integrations). Without a formally defined response contract, each handler risks inventing its own shape, leading to inconsistent error handling on the client, brittle parsing logic, and friction for AI agents or automated tooling that interact with the API.

This ADR establishes the canonical response envelope, error format, pagination strategy, HTTP status code conventions, SSE event schema, and URL structure for all API endpoints served by Opus Server.

> **Note for AI agents and automated tooling:** This ADR is the authoritative and complete specification of the Opus API contract. Do not infer or hallucinate response shapes, field names, or status codes beyond what is defined here. When in doubt, refer to this document.

---

## 2. Decision

Opus Server adopts a **uniform JSON response envelope** inspired by JSON:API, **RFC 7807 Problem Details** for error responses, **cursor-based pagination**, **strict REST HTTP status codes**, and a **versioning-free URL structure**.

---

### 2.1 Response Envelope

Every API response — success or error — is wrapped in a consistent top-level envelope:

```json
{
  "data": <object | array | null>,
  "error": <ProblemDetails | null>,
  "meta": <object | null>
}
```

**Rules:**

- On **success**: `data` carries the payload; `error` is `null`.
- On **error**: `error` carries the Problem Details object; `data` is `null`.
- `meta` is always present on paginated responses; `null` on non-paginated responses.
- `data` and `error` are **mutually exclusive** — never both non-null in the same response.

**Success example (single resource):**

```json
{
  "data": {
    "id": "agt_01HZ9XYZ",
    "name": "Daily Digest",
    "status": "running"
  },
  "error": null,
  "meta": null
}
```

**Success example (collection):**

```json
{
  "data": [
    { "id": "agt_01HZ9XYZ", "name": "Daily Digest", "status": "running" },
    { "id": "agt_02ABCDEF", "name": "Mail Sorter",  "status": "idle" }
  ],
  "error": null,
  "meta": {
    "cursor": {
      "next": "eyJpZCI6ImFndF8wMkFCQ0RFRiJ9",
      "has_more": true
    },
    "total": null,
    "request_id": "req_7fGh3kLm"
  }
}
```

---

### 2.2 Error Format — RFC 7807 Problem Details

Error responses conform to [RFC 7807](https://www.rfc-editor.org/rfc/rfc7807) wrapped inside the envelope's `error` field.

**Shape:**

```json
{
  "data": null,
  "error": {
    "type": "https://opus.local/errors/<error-slug>",
    "title": "<Human-readable summary>",
    "status": <HTTP status code>,
    "detail": "<Specific description of this occurrence>",
    "instance": "<Request path that produced the error>"
  },
  "meta": null
}
```

| Field | Type | Description |
|---|---|---|
| `type` | `string` (URI) | Stable identifier for the error category. Always `https://opus.local/errors/<slug>`. |
| `title` | `string` | Short, human-readable summary of the error type. Does not change per occurrence. |
| `status` | `integer` | HTTP status code mirrored from the response header. |
| `detail` | `string` | Occurrence-specific description. Safe to display to end users. |
| `instance` | `string` | Request URI that triggered the error. Aids in log correlation. |

**Registered error slugs:**

| Slug | Status | Description |
|---|---|---|
| `bad-request` | 400 | Malformed request body or query parameter |
| `unauthorized` | 401 | Missing or invalid authentication token |
| `forbidden` | 403 | Authenticated but insufficient permissions |
| `not-found` | 404 | Requested resource does not exist |
| `unprocessable-entity` | 422 | Request is well-formed but fails domain validation |
| `internal-server-error` | 500 | Unhandled server-side error |

**Error example (404):**

```json
{
  "data": null,
  "error": {
    "type": "https://opus.local/errors/not-found",
    "title": "Resource Not Found",
    "status": 404,
    "detail": "Agent with ID agt_01HZ9XYZ does not exist.",
    "instance": "/agents/agt_01HZ9XYZ"
  },
  "meta": null
}
```

**Error example (422 — validation failure):**

```json
{
  "data": null,
  "error": {
    "type": "https://opus.local/errors/unprocessable-entity",
    "title": "Validation Failed",
    "status": 422,
    "detail": "Field 'tick_interval' must be a valid Go duration string (e.g. '60s', '5m').",
    "instance": "/agents"
  },
  "meta": null
}
```

---

### 2.3 Pagination — Cursor-Based

All collection endpoints that may return more than one page of results use **opaque cursor-based pagination**. Offset-based pagination is not used.

**Rationale:** Agent logs and workflow runs are high-volume, append-only data streams. Cursor-based pagination is stable under concurrent inserts and does not suffer from the page-drift problem inherent in offset pagination.

**Request parameters:**

| Query Parameter | Type | Description |
|---|---|---|
| `cursor` | `string` (optional) | Opaque cursor returned by the previous page's `meta.cursor.next`. Omit to fetch the first page. |
| `limit` | `integer` (optional) | Number of items per page. Default: `20`. Maximum: `100`. |

**Pagination meta block:**

```json
"meta": {
  "cursor": {
    "next": "<opaque base64 string | null>",
    "has_more": true
  },
  "total": null,
  "request_id": "req_7fGh3kLm"
}
```

| Field | Type | Description |
|---|---|---|
| `cursor.next` | `string \| null` | Cursor to pass as `?cursor=` in the next request. `null` when no more pages exist. |
| `cursor.has_more` | `boolean` | Convenience flag. `true` when `cursor.next` is non-null. |
| `total` | `integer \| null` | Total item count. `null` when a count query is not feasible (e.g. live log streams). |
| `request_id` | `string` | Unique identifier for this request. Used for log correlation and client-side deduplication. |

**Example paginated request:**

```
GET /agents?limit=20
GET /agents?cursor=eyJpZCI6ImFndF8wMkFCQ0RFRiJ9&limit=20
```

---

### 2.4 HTTP Status Code Convention

Opus Server uses strict REST semantics for all HTTP status codes.

| Method | Scenario | Status Code |
|---|---|---|
| `GET` | Success | `200 OK` |
| `POST` | Resource created | `201 Created` |
| `PUT` / `PATCH` | Resource updated | `200 OK` |
| `DELETE` | Resource deleted | `204 No Content` (empty body) |
| Any | Bad request body | `400 Bad Request` |
| Any | Missing / invalid token | `401 Unauthorized` |
| Any | Insufficient permissions | `403 Forbidden` |
| Any | Resource not found | `404 Not Found` |
| Any | Domain validation failure | `422 Unprocessable Entity` |
| Any | Server error | `500 Internal Server Error` |

**Rules:**

- `204 No Content` responses have **no body** — not even the envelope.
- `500` responses always include the envelope with `error.type` set to `https://opus.local/errors/internal-server-error`. Internal stack traces are never exposed in `detail`.
- `401` is returned when authentication is absent or the token is expired. `403` is returned when the token is valid but the caller lacks permission.

---

### 2.5 URL Structure

Opus Server does **not** use API version prefixes in the URL path.

```
# Correct
GET /agents
GET /agents/{id}
GET /vault/entries

# Incorrect — version prefix is not used
GET /api/v1/agents
GET /v1/agents
```

**Rationale:** Opus is a self-hosted, single-tenant system. The server and the Dash client are deployed and upgraded together as a unit. URL versioning introduces coordination overhead (maintaining multiple active versions) that provides no benefit in this deployment model. Breaking changes are managed through the ADR process and release notes, not through parallel URL namespaces.

> **Note for AI agents:** Do not prepend `/api/`, `/v1/`, `/v2/`, or any version segment to Opus API paths.

**URL conventions:**

- Resource collections: `/{resource}` (plural noun)
- Single resource: `/{resource}/{id}`
- Nested resource: `/{resource}/{id}/{sub-resource}`
- Maximum nesting depth: two levels (e.g. `/agents/{id}/logs`)

---

### 2.6 SSE Event Schema

Opus Server streams real-time agent log events over Server-Sent Events (SSE). All SSE events conform to the following envelope.

**SSE connection endpoint:**

```
GET /agents/{id}/logs/stream
```

**SSE event wire format:**

```
event: agent.log
data: {"event":"agent.log","agent_id":"agt_01HZ9XYZ","sequence":42,"payload":{...},"timestamp":"2026-05-17T08:00:00Z"}

event: agent.status
data: {"event":"agent.status","agent_id":"agt_01HZ9XYZ","sequence":43,"payload":{"status":"completed"},"timestamp":"2026-05-17T08:00:01Z"}

event: heartbeat
data: {"event":"heartbeat","agent_id":"agt_01HZ9XYZ","sequence":null,"payload":null,"timestamp":"2026-05-17T08:00:30Z"}
```

**SSE envelope fields:**

| Field | Type | Description |
|---|---|---|
| `event` | `string` | Event type identifier (see registered events below). |
| `agent_id` | `string` | ID of the agent that produced the event. |
| `sequence` | `integer \| null` | Monotonically increasing sequence number per agent. Used for gap detection and client-side deduplication. `null` for `heartbeat` events. |
| `payload` | `object \| null` | Event-specific data. Shape varies by event type (see below). |
| `timestamp` | `string` (ISO 8601 UTC) | Server-side event timestamp. |

**Registered SSE event types:**

| Event | Description | Payload Shape |
|---|---|---|
| `agent.log` | A log line emitted by the agent runtime. | `{ "level": "info\|warn\|error", "message": "string" }` |
| `agent.status` | Agent lifecycle state change. | `{ "status": "idle\|running\|completed\|errored" }` |
| `agent.tool_call` | Agent invoked a tool. | `{ "tool": "string", "input": {}, "output": {} }` |
| `heartbeat` | Keep-alive ping emitted every 30 seconds. | `null` |

**SSE error handling:**

If the agent ID does not exist or the caller is not authorised, the server closes the SSE connection immediately after sending a single error event:

```
event: error
data: {"event":"error","agent_id":"agt_01HZ9XYZ","sequence":null,"payload":{"type":"https://opus.local/errors/not-found","title":"Resource Not Found","status":404,"detail":"Agent with ID agt_01HZ9XYZ does not exist.","instance":"/agents/agt_01HZ9XYZ/logs/stream"},"timestamp":"2026-05-17T08:00:00Z"}
```

Clients must handle the `error` event type and not attempt reconnection for `404` and `403` errors.

---

### 2.7 Go Implementation — Fiber Response Helpers

Centralised response helpers are defined in `delivery/http/response/` to enforce the envelope consistently across all handlers.

```go
// delivery/http/response/response.go
package response

import (
    "fmt"
    "github.com/gofiber/fiber/v2"
)

type Envelope[T any] struct {
    Data  T            `json:"data"`
    Error *Problem     `json:"error"`
    Meta  *Meta        `json:"meta"`
}

type Problem struct {
    Type     string `json:"type"`
    Title    string `json:"title"`
    Status   int    `json:"status"`
    Detail   string `json:"detail"`
    Instance string `json:"instance"`
}

type Meta struct {
    Cursor    *CursorMeta `json:"cursor,omitempty"`
    Total     *int64      `json:"total"`
    RequestID string      `json:"request_id"`
}

type CursorMeta struct {
    Next    *string `json:"next"`
    HasMore bool    `json:"has_more"`
}

func OK[T any](c *fiber.Ctx, data T) error {
    return c.Status(fiber.StatusOK).JSON(Envelope[T]{Data: data})
}

func Created[T any](c *fiber.Ctx, data T) error {
    return c.Status(fiber.StatusCreated).JSON(Envelope[T]{Data: data})
}

func NoContent(c *fiber.Ctx) error {
    return c.SendStatus(fiber.StatusNoContent)
}

func Paginated[T any](c *fiber.Ctx, data T, meta *Meta) error {
    return c.Status(fiber.StatusOK).JSON(Envelope[T]{Data: data, Meta: meta})
}

func Error(c *fiber.Ctx, status int, slug, title, detail string) error {
    return c.Status(status).JSON(Envelope[any]{
        Error: &Problem{
            Type:     fmt.Sprintf("https://opus.local/errors/%s", slug),
            Title:    title,
            Status:   status,
            Detail:   detail,
            Instance: c.Path(),
        },
    })
}
```

**Handler usage example:**

```go
// delivery/http/handler/agent_handler.go
func (h *AgentHandler) GetAgent(c *fiber.Ctx) error {
    id := c.Params("id")
    agent, err := h.service.FindByID(c.Context(), id)
    if err != nil {
        return response.Error(c, fiber.StatusNotFound, "not-found", "Resource Not Found",
            fmt.Sprintf("Agent with ID %s does not exist.", id))
    }
    return response.OK(c, agent)
}
```

---

### 2.8 TypeScript Client Types — Opus Dash

The response envelope is typed in `shared/types/api.ts` in the Dash frontend, consumed by all TanStack Query `queryFn` implementations.

```typescript
// shared/types/api.ts

export interface ApiEnvelope<T> {
  data: T | null;
  error: ProblemDetail | null;
  meta: PaginationMeta | null;
}

export interface ProblemDetail {
  type: string;
  title: string;
  status: number;
  detail: string;
  instance: string;
}

export interface PaginationMeta {
  cursor: {
    next: string | null;
    has_more: boolean;
  } | null;
  total: number | null;
  request_id: string;
}
```

```typescript
// shared/lib/api-client.ts (updated)
export const apiClient = {
  get: async <T>(path: string): Promise<T> => {
    const res = await fetch(`${BASE_URL}${path}`, {
      headers: { Authorization: `Bearer ${getToken()}` },
    });
    const envelope: ApiEnvelope<T> = await res.json();
    if (!res.ok || envelope.error) {
      throw envelope.error;
    }
    return envelope.data as T;
  },
};
```

---

## 3. Alternatives Considered

### 3.1 Naked Response (No Envelope)

Return the resource or array directly without a wrapper. Rejected because:

- Distinguishing success from error requires inspecting both HTTP status and response body shape independently
- No consistent place to attach pagination metadata or request IDs without ad-hoc HTTP response headers
- Inconsistent for AI agents and automated tooling that parse responses programmatically

### 3.2 JSON:API Full Specification

Full JSON:API compliance with `relationships`, `included`, `links`, and `jsonapi` top-level keys. Rejected because:

- Significantly increases response payload size and client parsing complexity
- JSON:API's relationship model does not map cleanly to Opus's domain model
- The envelope defined in this ADR captures the essential benefits (consistency, error shape, meta) without the overhead

### 3.3 GraphQL

Single flexible query endpoint. Rejected because:

- SSE streaming for agent logs does not fit naturally into the GraphQL subscription model without additional infrastructure (WebSocket server, subscription manager)
- Adds significant server implementation complexity with no benefit given Opus Dash's well-defined, stable query surface
- Go Fiber's strengths are in REST; GraphQL would require a separate runtime layer

### 3.4 URL Versioning (`/api/v1/`)

Adding a version prefix to all routes. Rejected because:

- Opus is a co-deployed monolith; the server and client are always upgraded together
- URL versioning implies maintaining multiple live API versions simultaneously, which is not a requirement for a self-hosted single-tenant system
- Breaking changes are communicated through the ADR process and release notes

---

## 4. Consequences

### 4.1 Positive

- **Consistency** — Every API response has a predictable shape; client parsing logic is centralised in `apiClient` and applied uniformly
- **AI agent compatibility** — The versioning-free URL structure and explicit schema defined in this ADR prevent hallucinated endpoints or version prefixes by automated tooling
- **RFC 7807 compliance** — Error responses are machine-readable and interoperable with standard HTTP tooling (curl, Insomnia, Postman)
- **Cursor pagination stability** — Paginated results remain stable under concurrent inserts; no page-drift on high-velocity agent log streams
- **SSE sequence numbers** — Monotonically increasing `sequence` fields enable gap detection and client-side deduplication without additional infrastructure
- **Type safety end-to-end** — `ApiEnvelope<T>` in TypeScript and `Envelope[T]` in Go provide compile-time guarantees at both ends of the contract

### 4.2 Negative / Trade-offs

- **Envelope overhead** — Wrapping every response adds a small payload overhead (`data`, `error`, `meta` keys) relative to naked responses; negligible over localhost
- **`204 No Content` exception** — `DELETE` responses deliberately break the envelope convention (no body); clients must handle this as a special case
- **Cursor opacity** — Cursors are opaque base64 strings; clients cannot inspect or construct them manually. This is by design but limits debugging transparency
- **No URL versioning escape hatch** — If a future breaking API change cannot be avoided, a new ADR will be required to define the migration strategy, as no versioning mechanism is in place

---

## 5. References

- [ADR-001: Server Clean Architecture](./ADR-001-server-clean-architecture.md)
- [ADR-003: Opus Dash Frontend Architecture](./ADR-003-dash-frontend-architecture.md)
- [RFC 7807 — Problem Details for HTTP APIs](https://www.rfc-editor.org/rfc/rfc7807)
- [JSON:API Specification](https://jsonapi.org)
- [Server-Sent Events — MDN](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)
- [TanStack Query — queryOptions](https://tanstack.com/query/latest/docs/framework/react/reference/queryOptions)
- [GoFiber — gofiber.io](https://gofiber.io)