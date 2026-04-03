# Audit Log Microservice — Design Spec

**Date:** 2026-04-02  
**Status:** Approved

---

## Overview

A centralized audit log microservice that provides an append-only, immutable record of all significant events across the platform. Built in Go with a gRPC interface and PostgreSQL persistence. Enforces immutability at the database role level (not the application layer). Supports compensation events for reversals — no updates or deletes, ever.

---

## Architecture

The service follows **Hexagonal (Clean) Architecture** as defined by Alex the Architect. Dependencies point inward: domain has no external dependencies; use cases define interfaces (ports); infrastructure implements them (adapters).

```
grpcapi → usecases → domain
              ↑ (interface only)
         persistence
              ↑
            infra  (wires everything; nothing imports infra directly)
```

### Project Structure

```
audit-log/
├── cmd/
│   └── server/
│       ├── main.go                     — entrypoint, graceful shutdown
│       └── wire/                       — Wire DI providers + wire_gen.go
│
├── internal/
│   ├── auditlog/                       — single domain
│   │   ├── domain/                     — pure domain: AuditEvent, enums, builders, errors
│   │   ├── grpcapi/                    — thin gRPC handlers (parse proto → call use case → respond)
│   │   │   └── internal/              — request/response mapping helpers (unexported)
│   │   ├── usecases/                   — business logic + port definitions
│   │   │   ├── audit_service.go        — implementation
│   │   │   ├── api.go                  — AuditService interface + //go:generate mockgen
│   │   │   └── repository_port.go     — EventStore interface + //go:generate mockgen
│   │   └── persistence/               — GORM repository implementing EventStore
│   │       └── internal/              — GORM model structs (unexported, persistence-private)
│   │
│   └── infra/
│       ├── grpcserver/                — gRPC server setup, interceptor chain
│       ├── config/                    — Viper config, sync.Once singleton
│       ├── node/                      — node ID, IP, version (sync.Once singleton)
│       ├── database/                  — GORM+pgx connection, AutoMigrate bootstrap
│       └── jsonpatch/                 — RFC 6902 diff computation (pure, no infra deps)
│
├── proto/                              — audit_log.proto + generated pb.go / grpc.pb.go
├── test/
│   ├── unit/doubles/auditlog/         — mockgen-generated mocks
│   ├── functional/                    — Godog BDD tests (driver + features + steps)
│   └── contract/provider/             — Pact provider verification
├── arch-go.yml                        — dependency rules enforced by arch-go
├── justfile                            — all developer commands
├── docker-compose.yml                  — postgres + otel-collector + audit-log service
└── .env.example                        — documented environment variables
```

---

## Domain Model

### AuditEvent

Defined in `internal/auditlog/domain/audit_event.go`. Pure Go — zero external imports.

**Static fields** (indexed, filterable):

| Field | Type | Notes |
|---|---|---|
| `ID` | UUID v7 | Generated server-side; time-sortable |
| `TenantID` | string | |
| `ActorID` | string | No PII — ID only |
| `ActorType` | `ActorType` enum | `User`, `Service`, `System` |
| `EntityType` | string | |
| `EntityID` | string | |
| `Action` | `Action` enum | `Created`, `Updated`, `Deleted`, `Compensated` |
| `Outcome` | `Outcome` enum | `Success`, `Failure`, `Partial` |
| `ServiceName` | string | |
| `SourceIP` | string | |
| `SessionID` | string | |
| `CorrelationID` | string | |
| `TraceID` | string | |
| `Timestamp` | `time.Time` | Always UTC; set server-side |
| `CompensatesID` | `*uuid.UUID` | Non-nil only for compensation events |

**Dynamic fields** (JSONB):

| Field | Type | Notes |
|---|---|---|
| `Before` | `map[string]any` | Entity state before action |
| `After` | `map[string]any` | Entity state after action |
| `Diff` | `map[string]any` | RFC 6902 JSON Patch — computed server-side from Before/After |
| `Metadata` | `map[string]any` | Arbitrary KV bag |
| `Reason` | string | Human justification |
| `Tags` | `[]string` | Free-form labels |

### Invariants Enforced by the Builder

- `ID` always generated server-side (UUID v7) — callers cannot supply it
- `Timestamp` always set to `time.Now().UTC()` server-side — callers cannot supply it
- `Diff` is **not** set by the builder — it is computed by the use case layer (after `Build()`, before `Save()`) using `infra/jsonpatch`, keeping domain free of infrastructure imports
- `CompensatesID` is required and validated when `Action == Compensated`
- Builder follows the action-slice pattern: only an `actions []auditEventBuilderHandler` slice; no state fields on the builder struct

### Enums

```go
type ActorType string
const (
    ActorTypeUser    ActorType = "user"
    ActorTypeService ActorType = "service"
    ActorTypeSystem  ActorType = "system"
)

type Action string
const (
    ActionCreated     Action = "CREATED"
    ActionUpdated     Action = "UPDATED"
    ActionDeleted     Action = "DELETED"
    ActionCompensated Action = "COMPENSATED"
)

type Outcome string
const (
    OutcomeSuccess Outcome = "SUCCESS"
    OutcomeFailure Outcome = "FAILURE"
    OutcomePartial Outcome = "PARTIAL"
)
```

---

## gRPC API

Defined in `proto/audit_log.proto`.

```protobuf
service AuditLog {
  rpc WriteEvent(WriteEventRequest) returns (WriteEventResponse);
  rpc WriteCompensation(WriteCompensationRequest) returns (WriteCompensationResponse);
  rpc QueryEvents(QueryEventsRequest) returns (QueryEventsResponse);
  rpc GetEvent(GetEventRequest) returns (GetEventResponse);
}
```

### WriteEvent

Caller supplies all static fields and optional `before`/`after` JSONB. Server generates `id`, `timestamp`, and `diff`. Returns the stored event with generated fields populated.

### WriteCompensation

Same input shape as `WriteEvent` plus required `compensates_id`. Server validates:
1. The referenced event exists
2. It belongs to the same `tenant_id`

Sets `action = COMPENSATED`. Returns the stored compensation event.

### QueryEvents

Filter by any combination of static fields. Paginated using cursor-based pagination: `page_token` (UUID v7 `id` of the last item seen) + `page_size`. Returns a `next_page_token` when more results exist.

### GetEvent

Fetch a single event by `id`. Returns `NOT_FOUND` if the event does not exist.

### Interceptor Chain

Three interceptors, applied to every method:

| Interceptor | Status | Purpose |
|---|---|---|
| Auth | Placeholder (no-op) | Slot for mTLS / JWT validation |
| Tracing | Active | `otelgrpc` — OTel spans + B3 context extraction from gRPC metadata |
| Metrics | Active | OTel metrics — request count and duration per method |

---

## Use Cases Layer

**`AuditService` interface** (`usecases/api.go`):

```go
type AuditService interface {
    WriteEvent(ctx context.Context, opts WriteEventOptions) (*domain.AuditEvent, error)
    WriteCompensation(ctx context.Context, opts WriteCompensationOptions) (*domain.AuditEvent, error)
    QueryEvents(ctx context.Context, opts QueryEventsOptions) (*QueryEventsResult, error)
    GetEvent(ctx context.Context, id uuid.UUID) (*domain.AuditEvent, error)
}
```

Options structs used for all multi-parameter methods (Alex's Service Interface Design rule):

- `WriteEventOptions` — all caller-supplied fields (static + dynamic, excluding id/timestamp/diff)
- `WriteCompensationOptions` — same as `WriteEventOptions` plus `CompensatesID uuid.UUID`
- `QueryEventsOptions` — all filterable fields (all optional) + `PageToken *uuid.UUID` + `PageSize int`
- `QueryEventsResult` — wraps `Events []*domain.AuditEvent` + `NextPageToken *uuid.UUID` (nil when no more pages)

**`EventStore` interface** (`usecases/repository_port.go`):

```go
type EventStore interface {
    Save(ctx context.Context, event *domain.AuditEvent) error
    FindByID(ctx context.Context, id uuid.UUID) (*domain.AuditEvent, error)
    Query(ctx context.Context, opts QueryEventsOptions) ([]*domain.AuditEvent, error)
}
```

Only the methods actually needed by the current use cases — no speculative additions.

---

## Persistence Layer

- GORM with `gorm.io/driver/postgres` (pgx under the hood)
- GORM model in `persistence/internal/audit_event_record.go` — unexported, private to the package
- Mapping functions `toRecord()` and `toDomain()` in the repository file
- `QueryEventsOptions` filters translated to GORM `Where` clauses; JSONB queries via `db.Raw` where needed

### Database Bootstrap (on startup)

Two separate DB connections are used:

1. **Admin connection** (DSN: `AUDIT_LOG_DB_ADMIN_DSN`, full DDL privileges) — used only during startup bootstrap:
   - `db.AutoMigrate(&AuditEventRecord{})` — creates/updates the table schema
   - Raw `db.Exec()` calls with `IF NOT EXISTS`/`IF NOT EXISTS` guards for:
     - **Append-only role:** `CREATE ROLE IF NOT EXISTS audit_log_writer; GRANT INSERT, SELECT ON audit_events TO audit_log_writer; REVOKE UPDATE, DELETE ON audit_events FROM PUBLIC;`
     - **GIN indexes** on JSONB columns: `before`, `after`, `diff`, `metadata`
     - **Partial B-tree indexes** on: `tenant_id`, `actor_id`, `entity_type`, `action`, `timestamp`

2. **Runtime connection** (DSN: `AUDIT_LOG_DB_DSN`, `audit_log_writer` role) — used by the GORM repository for all application queries. This connection physically cannot UPDATE or DELETE — enforcing append-only at the database level, not the application layer.

> **Note:** Table partitioning (`PARTITION BY RANGE`) is deferred. It can be added later via `pg_partman` when data volume justifies it.

---

## Dependency Injection

Google Wire (`github.com/google/wire`) for compile-time DI. All providers in `cmd/server/wire/`. Wire binds:

- `*persistence.EventRepository` → `usecases.EventStore`
- `*usecases.SimpleAuditService` → `usecases.AuditService`

---

## Configuration

Viper with `sync.Once` singleton. Environment variable prefix: `AUDIT_LOG`. Config file: `config/server.yaml`.

Key variables:

| Variable | Default | Description |
|---|---|---|
| `AUDIT_LOG_SERVER_PORT` | `50051` | gRPC listen port |
| `AUDIT_LOG_SERVER_PPROF_PORT` | `6061` | pprof internal port |
| `AUDIT_LOG_DB_DSN` | — | Postgres DSN (audit_log_writer role, runtime) |
| `AUDIT_LOG_DB_ADMIN_DSN` | — | Postgres DSN (admin role, bootstrap only) |
| `AUDIT_LOG_OTEL_ENABLED` | `true` | Enable OTel export |
| `AUDIT_LOG_OTEL_ENDPOINT` | `localhost:4317` | OTLP gRPC endpoint |
| `AUDIT_LOG_OTEL_SERVICE_NAME` | `audit-log` | OTel service name |
| `AUDIT_LOG_OTEL_ENVIRONMENT` | `development` | Deployment environment |
| `AUDIT_LOG_OTEL_SAMPLE_RATE` | `0.1` | Trace sample rate (production) |
| `AUDIT_LOG_GENERAL_LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |

---

## Observability

### Logging

- `log/slog` with a custom OTEL log handler that auto-injects `trace_id` and `span_id`
- JSON in production, text in development
- Context-aware logger propagated via `context.Context`
- No `fmt.Printf` or `log.Printf` anywhere in production code
- gRPC errors mapped: `NOT_FOUND`, `INVALID_ARGUMENT` → `Info/Warn`; unhandled failures → `Error`

### Metrics

OTel Metrics API + OTLP export. All metrics defined at package init:

- `audit_log.grpc.requests.total{method, status}` — counter
- `audit_log.grpc.request.duration.seconds{method, status}` — histogram
- `audit_log.events.written.total{outcome}` — counter
- `audit_log.compensations.written.total{outcome}` — counter

No high-cardinality labels (`tenant_id`, `actor_id` go in spans, not metric labels).

### Tracing

- OTel SDK + OTLP export + **B3 multi-header** propagation (mandatory convention)
- `otelgrpc` unary + stream interceptors on the gRPC server
- Spans for: each gRPC method, `AuditService.*`, `EventRepository.*`, diff computation
- Span attributes: `tenant_id`, `actor_id`, `entity_type`, `action` — never PII
- Errors recorded with `span.RecordError` + `codes.Error`
- Sampling: 100% dev/staging, 10% parent-based production

### Profiling

- `net/http/pprof` on `:6061` (internal-only)

---

## Testing Strategy

Following Andy the QA's blueprint.

### Unit Tests — Ginkgo v2 + Gomega + gomock

- Dot imports: `import . "github.com/onsi/ginkgo/v2"` and `. "github.com/onsi/gomega"`
- Mock library: `go.uber.org/mock/gomock`
- `//go:generate mockgen` on `usecases/api.go` and `usecases/repository_port.go`
- Mocks output: `test/unit/doubles/auditlog/`
- Structure: `Describe("AuditService") → Context("WriteEvent") → When("...") → It("...")`
- Mock expectations declared in `When`/`BeforeEach` only — never at `Describe` or `Context` level
- All test packages use `_test` suffix (e.g., `package usecases_test`)

### Mutation Testing — gremlins

- `gremlins unleash ./internal/auditlog/domain/... ./internal/auditlog/usecases/...`
- Target: `domain/` ≥ 90%, `usecases/` ≥ 85%
- Surviving mutants in business logic are a merge blocker

### Functional Tests — Godog (BDD)

- `test/functional/` — `driver/` (gRPC client wrapper), `features/`, `steps/`
- All four gRPC methods covered: `WriteEvent`, `WriteCompensation`, `QueryEvents`, `GetEvent`
- Runs against a live stack (Postgres via docker-compose)
- `just functional tags="~@pending"`

### Contract Tests — Pact v4 (gRPC plugin)

- This service is a **provider**
- Consumer services publish contracts to a Pact Broker
- Provider verification in `test/contract/provider/`
- Provider states seed Postgres before each interaction is verified
- Runs in CI on every provider change

### Coverage Targets

| Layer | Target |
|---|---|
| `domain/` | 90%+ |
| `usecases/` | 85%+ |
| `persistence/` | 60%+ |
| `grpcapi/` | 60%+ |
| `infra/` | 40%+ |

### Quality Gate (all three must pass before merge)

1. `just unit` — unit tests + coverage threshold
2. `just arch` — arch-go at 100% compliance
3. `just functional` — BDD scenarios against live stack

---

## Architecture Dependency Rules (arch-go.yml)

```yaml
dependenciesRules:
  - package: audit-log/internal/auditlog/grpcapi
    shouldNotDependsOn:
      internal:
        - audit-log/internal/auditlog/persistence
  - package: audit-log/internal/auditlog/usecases
    shouldNotDependsOn:
      internal:
        - audit-log/internal/auditlog/persistence
        - audit-log/internal/auditlog/grpcapi
  - package: audit-log/internal/auditlog/domain
    shouldNotDependsOn:
      internal:
        - audit-log/internal/auditlog/usecases
        - audit-log/internal/auditlog/persistence
        - audit-log/internal/auditlog/grpcapi
        - audit-log/internal/infra
  - package: audit-log/internal/infra
    shouldNotDependsOn:
      internal:
        - audit-log/internal/auditlog
```

---

## Justfile Commands

```
just build          — build the binary
just run            — build + start docker-compose + run
just wire           — regenerate Wire DI code
just mock           — regenerate mocks (go generate ./internal/...)
just arch           — validate architecture rules (arch-go)
just unit           — run unit tests with race detection + coverage
just functional     — build + start stack + run Godog scenarios
just migrate        — run database bootstrap (role + indexes + AutoMigrate)
just release v=...  — tag and push release
```

---

## docker-compose Services

| Service | Port | Purpose |
|---|---|---|
| `postgres` | `5432` | Primary database |
| `otel-collector` | `4317`, `4318` | OTLP receiver |
| `audit-log` | `50051`, `6061` | gRPC API + pprof |

---

## Out of Scope (for now)

- OpenSearch / CDC dual-write — `EventStore` interface is designed for a future decorator to fan out; use case layer requires no changes
- mTLS — auth interceptor is a wired no-op placeholder
- Table partitioning — deferred until data volume justifies it; `pg_partman` is the path forward
- Metrics scraping endpoint — OTel collector handles export; no standalone `/metrics` HTTP endpoint
