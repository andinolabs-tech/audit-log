# Embedded SPA + HTTP Query API — Design Spec

**Date:** 2026-04-28  
**Status:** Approved

---

## Overview

Add an HTTP server to the existing `audit-log` binary (which currently runs only gRPC on port 50051). The HTTP server serves an embedded React SPA and exposes two query endpoints:

- `GET /api/events` — mirrors the gRPC `QueryEvents` method; supports filtering by multiple namespaces
- `GET /api/namespaces` — returns the distinct list of namespaces stored in the database

The UI is read-only: a multi-select namespace picker (populated from `/api/namespaces`) + Search button + results list. No create or delete.

---

## Architecture

Follows the existing hexagonal architecture. The new `httpapi` package sits at the same layer as `grpcapi`:

```
httpapi → usecases → domain
grpcapi → usecases → domain
               ↑ (interface only)
          persistence
```

New packages:
```
internal/auditlog/httpapi/
  handler.go                   ← thin HTTP handlers (events + namespaces)
  internal/mapconv/mapconv.go  ← URL params → QueryEventsOptions, domain → JSON

internal/web/
  embed.go   ← //go:embed all:dist, Handler() func
  dist/      ← built by vite (gitignored)
```

---

## Domain / Use-case Changes

### `QueryEventsOptions.Namespace` → `Namespaces`

`Namespace *string` is replaced by `Namespaces []string` to support filtering by one or more namespaces at once.

- **Persistence**: when `Namespaces` is non-empty, the query uses `WHERE namespace IN (?)` instead of `= ?`.
- **gRPC mapconv**: maps the single proto `optional string namespace` field to a 1-element slice (`Namespaces: []string{ns}`) when the field is set — fully backward compatible.
- **HTTP mapconv**: collects all `?namespace=` repeated query params into the slice.

### New `ListNamespaces` operation

Added to both `EventStore` and `AuditService`:

```go
// EventStore
QueryNamespaces(ctx context.Context) ([]string, error)

// AuditService
ListNamespaces(ctx context.Context) ([]string, error)
```

`EventRepository.QueryNamespaces` executes:
```sql
SELECT DISTINCT namespace FROM audit_events
WHERE namespace IS NOT NULL AND namespace != ''
ORDER BY namespace ASC
```

---

## HTTP API

### `GET /api/events`

#### Query Parameters (all optional)

| Param          | Repeatable | Maps to                             |
|----------------|------------|-------------------------------------|
| `tenant_id`    | no         | `QueryEventsOptions.TenantID`       |
| `namespace`    | **yes**    | `QueryEventsOptions.Namespaces`     |
| `actor_id`     | no         | `QueryEventsOptions.ActorID`        |
| `actor_type`   | no         | `QueryEventsOptions.ActorType`      |
| `entity_type`  | no         | `QueryEventsOptions.EntityType`     |
| `entity_id`    | no         | `QueryEventsOptions.EntityID`       |
| `action`       | no         | `QueryEventsOptions.Action`         |
| `outcome`      | no         | `QueryEventsOptions.Outcome`        |
| `service_name` | no         | `QueryEventsOptions.ServiceName`    |
| `page_size`    | no         | `QueryEventsOptions.PageSize` (default 20) |
| `page_token`   | no         | `QueryEventsOptions.PageToken`      |

`namespace` is repeatable: `?namespace=auth&namespace=billing` filters events in either namespace.

#### Response (200 OK)

```json
{
  "events": [
    {
      "id": "uuid",
      "tenant_id": "...",
      "namespace": "...",
      "actor_id": "...",
      "actor_type": "user|service|system",
      "entity_type": "...",
      "entity_id": "...",
      "action": "CREATED|UPDATED|DELETED|COMPENSATED",
      "outcome": "SUCCESS|FAILURE|PARTIAL",
      "service_name": "...",
      "source_ip": "...",
      "session_id": "...",
      "correlation_id": "...",
      "trace_id": "...",
      "timestamp": "RFC3339",
      "occurred_at": "RFC3339|null",
      "compensates_id": "uuid|null",
      "reason": "...",
      "tags": [],
      "before": {},
      "after": {},
      "diff": {},
      "metadata": {}
    }
  ],
  "next_page_token": "uuid|empty"
}
```

---

### `GET /api/namespaces`

No parameters.

#### Response (200 OK)

```json
{
  "namespaces": ["auth", "billing", "orders"]
}
```

Returns an empty array when no events exist yet.

---

### Errors (both endpoints)

| Condition           | Status | Body               |
|---------------------|--------|--------------------|
| Invalid query param | 400    | `{"error": "..."}` |
| Invalid page_size   | 400    | `{"error": "..."}` |
| Internal error      | 500    | `{"error": "..."}` |

---

## Static File Serving + SPA Fallback (`internal/web/`)

- `//go:embed all:dist` (the `all:` prefix includes files starting with `_` or `.`)
- `fs.Sub` strips the `dist/` prefix
- Cache headers:
  - `/assets/*` → `Cache-Control: public, max-age=31536000, immutable`
  - `index.html` → `Cache-Control: no-cache`
- SPA fallback: if `fs.Stat` fails for the requested path, rewrite `r.URL.Path = "/"` and serve `index.html`

---

## HTTP Server Routing (in `main.go`)

```
GET  /api/events      → httpapi handler (QueryEvents)
GET  /api/namespaces  → httpapi handler (ListNamespaces)
/*                    → web.Handler() (embedded SPA + SPA fallback)
```

HTTP server runs on port 8080 (default), configurable via `AUDIT_LOG_HTTP_PORT` env or `http_port` in `config/server.yaml`.

---

## Config Changes

Add to `internal/infra/config/config.go`:
```go
HTTPPort int  // default 8080, env AUDIT_LOG_HTTP_PORT
```

---

## `main.go` Changes

Start HTTP server concurrently alongside gRPC. Graceful shutdown drains both:
- gRPC: `GracefulStop()` with 15s timeout
- HTTP: `Shutdown(ctx)` with 10s timeout

---

## Frontend (`web/`)

### Stack
- Vite 5 + React 18 + TypeScript + Tailwind CSS 3
- React Router v6

### File Structure

```
web/
  src/
    App.tsx           ← Router: / and /events both render QueryPage
    pages/
      QueryPage.tsx   ← namespace multi-select + page_size + Search button + results table
    types/
      event.ts        ← TypeScript AuditEvent type
  index.html
  package.json
  vite.config.ts      ← proxy /api → http://localhost:8080
  tailwind.config.ts
  tsconfig.json
  postcss.config.js
```

### Routes

| Route     | Component  | Purpose                               |
|-----------|------------|---------------------------------------|
| `/`       | QueryPage  | Primary entry point                   |
| `/events` | QueryPage  | Second route (validates SPA fallback) |

### UI Behaviour

1. On load: `GET /api/namespaces` populates a multi-select list (checkboxes or `<select multiple>`)
2. User selects one or more namespaces + sets page size (default 20)
3. Search button → `GET /api/events?namespace=a&namespace=b&page_size=<n>`
4. Results table columns: `timestamp`, `namespace`, `action`, `actor_id`, `entity_type`, `outcome`
5. "Load more" button if `next_page_token` is present
6. Empty state and error state handled

---

## `justfile` Changes

Add targets to the existing `justfile`:

```just
web-build:
    cd web && npm ci && npm run build

build: web-build
    go build -o bin/audit-log ./cmd/server
```

The existing `build` target currently runs only `go build`; it will be replaced to first build the frontend.

---

## `arch-go.yml` Addition

```yaml
- package: "audit-log/internal/auditlog/httpapi"
  shouldNotDependsOn:
    internal:
      - "audit-log/internal/auditlog/persistence"
```

---

## Wire

`httpapi.NewHandler(svc usecases.AuditService)` is instantiated directly in `main.go` using the same `AuditService` already wired by `wire.InitializeGRPC`. No changes to the wire graph are needed.

---

## Out of Scope

- Authentication / authorization on the HTTP API
- Write endpoints (POST, DELETE) via HTTP
