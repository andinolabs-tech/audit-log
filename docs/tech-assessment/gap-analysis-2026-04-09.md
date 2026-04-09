# Architecture Gaps — audit-log

> Expert review against Architecture, QA & Testing, Observability, Reliability, and Frontend. Reviewers: Alex (Architecture), Andy (QA), Kira (Observability), Solei (Reliability), Vela (Frontend). Expert selection: **all** (unscoped “technical assessment”).

---

## Alex — Architecture

### GAP-A1: Domain, API, and persistence are misaligned on required fields (🔴 Critical)

**What the code does:** The domain builder rejects events without a namespace, but the use case never sets it from incoming work. `WriteEventOptions` has no `Namespace` field, `mapconv.WriteEventRequestToOpts` does not map one, and the proto `WriteEventRequest` has no `namespace` field. `SimpleAuditService.writeAndSave` builds events without `WithNamespace`.

```38:61:internal/auditlog/usecases/audit_service.go
func (s *SimpleAuditService) writeAndSave(ctx context.Context, opts WriteEventOptions, compensates *uuid.UUID) (*domain.AuditEvent, error) {
	b := domain.NewAuditEventBuilder().
		WithTenantID(opts.TenantID).
		WithActorID(opts.ActorID).
		WithActorType(opts.ActorType).
		WithEntityType(opts.EntityType).
		WithEntityID(opts.EntityID).
		WithOutcome(opts.Outcome).
		WithServiceName(opts.ServiceName).
		// ... no WithNamespace
```

```188:193:internal/auditlog/domain/builder.go
	if e.TenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if e.Namespace == "" {
		return nil, ErrNamespaceRequired
	}
```

The persistence model also omits `Namespace` and `OccurredAt` while the domain entity defines them (`AuditEventRecord` vs `AuditEvent`), so even after fixing the API, storage and contract need to agree on which dimensions are queryable and immutable.

**Why it matters:** Every successful `WriteEvent` path must either fail consistently at the boundary with a clear validation error or accept and persist required dimensions. As written, callers cannot satisfy `ErrNamespaceRequired` through the public gRPC contract; unit tests fail and a running server would reject writes that tests expect to succeed.

**What it should be:** Add `namespace` (and, if required by the domain, `occurred_at`) to the proto, options struct, mapper, and GORM model/migrations together in one change; or relax domain rules if namespace is optional by product intent. Example direction for the use case chain:

```go
// WriteEventOptions — add field
Namespace string

// audit_service.go — wire builder
b := domain.NewAuditEventBuilder().
    WithTenantID(opts.TenantID).
    WithNamespace(opts.Namespace).
    // ...
```

---

### GAP-A2: Architecture tool enforces zero test coverage threshold (🟡 Medium)

**What the code does:** `arch-go.yml` sets `coverage: 0`, so dependency compliance is gated but coverage is not.

```1:4:arch-go.yml
version: 1
threshold:
  compliance: 100
  coverage: 0
```

**Why it matters:** Layer rules can pass while business logic has no meaningful test obligation, which weakens the “quality gate” story for merges.

**What it should be:** Raise `coverage` gradually (e.g. start with `internal/auditlog/domain` and `usecases`) to match team targets in Andy’s rubric, and keep `just unit` as the source of truth for enforcement.

---

## Andy — QA & testing

### GAP-Q1: Unit suite fails on main write/compensation paths (🔴 Critical)

**What the code does:** `go test ./internal/auditlog/usecases/...` fails three specs with `namespace is required`, consistent with GAP-A1.

**Why it matters:** A red unit suite blocks trust in CI and makes refactors unsafe. Andy’s rubric expects unit tests to be deterministic and green before merge.

**What it should be:** Fix the domain/API alignment (GAP-A1), then re-run `go test ./... -race -count=1` until green; optionally add a CI step that fails if `coverage.out` regresses below a floor.

---

### GAP-Q2: No mutation testing on business logic (🟡 Medium)

**What the code does:** There is no `gremlins` (or equivalent) configuration or `just` target; quality relies on line coverage only.

**Why it matters:** High line coverage with weak assertions leaves business rules under-protected; mutation testing catches assertion gaps early.

**What it should be:** Add a `just mutation` (or CI job) targeting `./internal/auditlog/domain/...` and `./internal/auditlog/usecases/...`, and track kill rate over time.

---

### GAP-Q3: No consumer-driven contract tests for the gRPC API (🟡 Medium)

**What the code does:** Functional tests speak gRPC directly with Godog; there is no Pact (or buf/schema registry) workflow for downstream consumers.

**Why it matters:** For a shared audit service, proto evolution without consumer verification risks silent breakage across teams.

**What it should be:** Introduce provider-side contract verification (e.g. Pact plugin for gRPC, or Buf breaking-change detection in CI) and publish/version protos explicitly.

---

## Kira — Observability

### GAP-K1: OTLP exporters always use insecure gRPC (🟠 High)

**What the code does:** Trace and metric exporters use `WithInsecure()` unconditionally.

```45:48:internal/infra/telemetry/telemetry.go
	traceExp, err := otlptrace.New(ctx, otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(cfg.OTelEndpoint),
		otlptracegrpc.WithInsecure(),
	))
```

**Why it matters:** Telemetry can carry operational metadata; on untrusted networks this is tampering and eavesdropping risk. Production should use TLS and explicit credential config.

**What it should be:** Gate insecure mode on `development` only; otherwise use `WithTLSCredentials` (or OTLP HTTP with TLS) driven by config:

```go
opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.OTelEndpoint)}
if cfg.OTelInsecure {
    opts = append(opts, otlptracegrpc.WithInsecure())
} else {
    opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")))
}
traceExp, err := otlptrace.New(ctx, otlptracegrpc.NewClient(opts...))
```

---

### GAP-K2: Service version in the OTel resource is hardcoded (🟡 Medium)

**What the code does:** `semconv.ServiceVersion("0.0.0-dev")` is fixed in code.

```34:39:internal/infra/telemetry/telemetry.go
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.OTelServiceName),
			semconv.DeploymentEnvironment(cfg.OTelEnvironment),
			semconv.ServiceVersion("0.0.0-dev"),
		),
	)
```

**Why it matters:** Incidents and deployments cannot be correlated to the actual binary or image tag.

**What it should be:** Inject version at link time (`-ldflags`) or read from `runtime/buildinfo` / env, matching the Docker image tag in CI.

---

### GAP-K3: No Prometheus `/metrics` scrape endpoint (🟡 Medium)

**What the code does:** Metrics flow through OTel SDK → OTLP gRPC only; there is no `promhttp` handler.

**Why it matters:** Kira’s checklist expects a scrape endpoint for Prometheus-style monitoring stacks; teams that do not ingest OTLP metrics lose visibility unless they add another sidecar path.

**What it should be:** Either expose OTel metrics via a Prometheus exporter bridge or document OTLP as the sole standard and provide Grafana/collector config as code in-repo.

---

## Solei — Reliability

### GAP-S1: gRPC surface has no authentication or authorization (🔴 Critical)

**What the code does:** The unary interceptor is explicitly a no-op placeholder.

```10:13:internal/infra/grpcserver/server.go
// authUnary is a placeholder for future mTLS / JWT validation.
func authUnary(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	return handler(ctx, req)
}
```

**Why it matters:** An audit log is a sensitive control-plane service. Unauthenticated gRPC allows any network peer to write or read events if the port is reachable.

**What it should be:** Enforce mTLS or bearer validation in `authUnary`, propagate identity to the use case layer for tenant-scoped authorization, and document required client credentials.

---

### GAP-S2: Database client has no explicit pool or statement timeouts (🟡 Medium)

**What the code does:** `OpenGORM` opens Postgres with default `gorm.Config{}` only.

```10:15:internal/infra/database/postgres.go
func OpenGORM(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}
	return db, nil
}
```

**Why it matters:** Without pool limits and timeouts, slow Postgres can exhaust connections or stall requests unbounded, conflicting with “every outbound call under a deadline.”

**What it should be:** Configure `sql.DB` `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`, and use `context`-bound queries; optionally set `statement_timeout` in the DSN for the app role.

---

### GAP-S3: No inbound rate limiting or load shedding (🟡 Medium)

**What the code does:** The gRPC server registers tracing/stats and a no-op auth interceptor only.

**Why it matters:** A burst of writers could overload Postgres or the process without a 429/RESOURCE_EXHAUSTED style back-pressure signal.

**What it should be:** Add token-bucket or max-in-flight limits with documented limits and metrics on shed events.

---

### GAP-S4: gRPC errors may leak raw internal details (🟠 High)

**What the code does:** Unknown use case errors become `Internal` with the raw Go error string.

```110:120:internal/auditlog/grpcapi/server.go
func mapUsecaseError(err error) error {
	switch {
	// ...
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
```

**Why it matters:** Database or infrastructure errors can expose schema or host details to clients.

**What it should be:** Log the full error server-side; return a stable `Internal` message to clients:

```go
default:
    slog.ErrorContext(ctx, "audit rpc failed", "err", err)
    return status.Error(codes.Internal, "internal error")
}
```

---

## Vela — Frontend

### GAP-V1: No first-party UI; consumer integration is undefined in-repo (🟢 Low)

**What the code does:** The repository is a Go gRPC service with protobuf definitions only. There is no React/Next.js app or BFF.

**Why it matters:** Not a defect for this repo, but product teams need a documented pattern (generated clients, versioning, auth headers) to avoid ad-hoc direct calls that bypass organizational standards.

**What it should be:** Add a short “Client integration” section to `README.md` (or internal docs): proto package version, breaking-change policy, and example stub for a BFF calling the service with mTLS/JWT.

---

## Priority summary

| ID     | Severity   | Area            | Gap |
|--------|------------|-----------------|-----|
| GAP-A1 | 🔴 Critical | Architecture    | Namespace (and related fields) required in domain but missing from API, use case, and persistence |
| GAP-Q1 | 🔴 Critical | QA & Testing    | Failing unit tests on write/compensation due to namespace mismatch |
| GAP-S1 | 🔴 Critical | Reliability     | No gRPC authentication/authorization on audit APIs |
| GAP-K1 | 🟠 High    | Observability   | OTLP clients always use insecure transport |
| GAP-S4 | 🟠 High    | Reliability     | Internal gRPC errors may expose raw `err.Error()` to clients |
| GAP-A2 | 🟡 Medium  | Architecture    | arch-go coverage threshold set to 0 |
| GAP-Q2 | 🟡 Medium  | QA & Testing    | No mutation testing for domain/usecases |
| GAP-Q3 | 🟡 Medium  | QA & Testing    | No consumer contract verification for gRPC/proto |
| GAP-K2 | 🟡 Medium  | Observability   | Hardcoded OTel service version |
| GAP-K3 | 🟡 Medium  | Observability   | No Prometheus `/metrics` scrape path |
| GAP-S2 | 🟡 Medium  | Reliability     | No explicit DB pool/timeout configuration |
| GAP-S3 | 🟡 Medium  | Reliability     | No inbound rate limiting / load shedding |
| GAP-V1 | 🟢 Low     | Frontend        | No UI; document consumer/BFF expectations |

---

## Coverage notes

Assessment focused on: `cmd/server`, `internal/auditlog/{domain,usecases,persistence,grpcapi}`, `internal/infra/{config,database,grpcserver,telemetry}`, `test/{functional,unit}`, `proto`, CI workflows (`ci-docker.yml`, `.drone.yml`), `arch-go.yml`, `justfile`, `Dockerfile`. Generated code (`wire_gen.go`, `*.pb.go`) was used for wiring verification only.

**Verification run (2026-04-09):** `go test ./... -count=1` — **failed** in `internal/auditlog/usecases` with `namespace is required` (see GAP-Q1).
