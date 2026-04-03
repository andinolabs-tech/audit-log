# audit-log

A Go service that stores and queries **audit events** over **gRPC**, backed by **PostgreSQL**, with **OpenTelemetry** tracing and metrics, and optional startup **schema migration** when an admin database URL is configured.

## Features

- **gRPC API** (`auditlog.v1.AuditLog`): write events, write compensating events, paginated query, and get-by-id.
- **PostgreSQL** persistence via GORM; optional admin DSN for `AutoMigrate` and bootstrap SQL (roles, indexes).
- **OpenTelemetry** (gRPC instrumentation, OTLP export); local stack includes an OpenTelemetry Collector.
- **Profiling**: `pprof` HTTP endpoint (configurable port, default `6061`).
- **gRPC reflection** enabled for tooling (e.g. `grpcurl`).

## Requirements

- **Go** 1.26.1 (see `go.mod` and `.mise.toml`)
- **Docker** (for Compose: Postgres + OTel Collector) when using `just run`
- **just** (optional) for task shortcuts; otherwise use the equivalent `go` / `docker compose` commands below
- **protoc** + Go plugins only if you regenerate protobuf code (`just proto`)

## Quick start

1. Copy environment defaults and adjust if needed:

   ```bash
   cp .env.example .env
   ```

2. Start dependencies (Postgres and OTel Collector):

   ```bash
   docker compose up -d --wait
   ```

3. Run the server (from repo root):

   ```bash
   just run
   ```

   This builds `bin/audit-log`, brings Compose up, then runs the binary. Alternatively:

   ```bash
   go build -o bin/audit-log ./cmd/server
   ./bin/audit-log
   ```

The process listens for gRPC on **`50051`** by default and exposes **pprof** on **`6061`**.

## Configuration

Configuration is loaded from `server.yaml` (search paths: `.`, `./config`, `/etc/audit-log`) and overridden by environment variables. All env vars use the prefix **`AUDIT_LOG_`**.

| Key (YAML / env suffix) | Description |
|-------------------------|-------------|
| `db_dsn` / `DB_DSN` | **Required.** Runtime PostgreSQL DSN (e.g. writer role). |
| `db_admin_dsn` / `DB_ADMIN_DSN` | Optional. Superuser/admin DSN for migrations and bootstrap SQL on startup. |
| `server_port` / `SERVER_PORT` | gRPC listen port (default `50051`). |
| `server_pprof_port` / `SERVER_PPROF_PORT` | pprof HTTP port (default `6061`). |
| `otel_enabled` / `OTEL_ENABLED` | Enable OpenTelemetry (default `true`). |
| `otel_endpoint` / `OTEL_ENDPOINT` | OTLP gRPC endpoint (default `localhost:4317`). |
| `otel_service_name` / `OTEL_SERVICE_NAME` | Service name in telemetry (default `audit-log`). |
| `otel_environment` / `OTEL_ENVIRONMENT` | e.g. `development` (text logs) vs production-style JSON logging behavior in `main`. |
| `otel_sample_rate` / `OTEL_SAMPLE_RATE` | Trace sampling ratio (default `0.1`). |
| `general_log_level` / `GENERAL_LOG_LEVEL` | `debug`, `info`, `warn`, or `error` (default `info`). |

See `config/server.yaml` and `.env.example` for concrete values.

## gRPC API

Package: `auditlog.v1`. RPCs:

- `WriteEvent` — append an audit event (tenant, actor, entity, action, before/after payloads, metadata, tags, etc.).
- `WriteCompensation` — record a compensating event linked to a prior event id.
- `QueryEvents` — filter by optional dimensions and paginate with `page_token` / `page_size`.
- `GetEvent` — fetch a single event by id.

Protobuf definitions: `proto/auditlogv1/audit_log.proto`. Regenerate Go stubs with `just proto` (requires `protoc` and paths as in the `justfile`).

## Docker image

Build a static binary image:

```bash
docker build -t audit-log .
```

The image exposes ports **50051** (gRPC) and **6061** (pprof). Pass configuration via environment variables (e.g. `AUDIT_LOG_DB_DSN`).

## Development

| Command | Purpose |
|---------|---------|
| `just build` | Build `bin/audit-log`. |
| `just wire` | Regenerate Google Wire DI (`cmd/server/wire`). |
| `just mock` | Regenerate use case mocks. |
| `just unit` | Unit tests with race detector and coverage. |
| `just functional` | Godog/Cucumber functional tests. |
| `just arch` | Run [arch-go](https://github.com/arch-go/arch-go) architecture checks. |
| `just proto` | Regenerate protobuf Go code. |

Database migrations for production are expected to be applied according to your ops process; the server can run `AutoMigrate` when `AUDIT_LOG_DB_ADMIN_DSN` is set—see `cmd/server/main.go` and persistence bootstrap helpers.
