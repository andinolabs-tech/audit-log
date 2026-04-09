# In-Memory Database Fallback

**Date:** 2026-04-09  
**Status:** Approved

## Context

The service currently requires `AUDIT_LOG_DB_DSN` to be set at startup — it returns a fatal error if missing. This blocks running the service (and functional/integration tests against it) without a live PostgreSQL instance.

The goal is to make `DBDSN` optional: when it is not provided, the service starts with a SQLite in-memory database so it can be used for testing without any external infrastructure.

## Architecture

The change is isolated to the startup path. The repository layer, service layer, and gRPC server are untouched. `*gorm.DB` is constructed conditionally and handed to Wire — everything downstream sees no difference.

```
DBDSN == ""  →  database.OpenInMemory()  →  AutoMigrateModel()  ─┐
                                                                    ├─→  wire.InitializeGRPC(db)
DBDSN != ""  →  database.OpenGORM(dsn)  ─────────────────────────┘
```

`gorm.io/driver/sqlite` is already a direct dependency (`go.mod` line 26), so no new packages are introduced.

## Components

### 1. `internal/infra/config/config.go`

Remove the required validation for `DBDSN` (current lines 67–69):

```go
// Remove this block:
if global.DBDSN == "" {
    loadErr = fmt.Errorf("required configuration missing: AUDIT_LOG_DB_DSN")
}
```

`DBDSN` becomes optional. The `fmt` import is only used by this block and must be removed.

### 2. `internal/infra/database/sqlite.go` (new file)

```go
package database

import (
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

// OpenInMemory opens a SQLite in-memory GORM connection.
// The database is ephemeral — data is lost when the connection is closed.
func OpenInMemory() (*gorm.DB, error) {
    return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
}
```

### 3. `cmd/server/main.go`

Restructure the DB setup block to branch on `cfg.DBDSN`:

```go
var db *gorm.DB
if cfg.DBDSN == "" {
    slog.Warn("AUDIT_LOG_DB_DSN not set — using in-memory SQLite (data will not persist)")
    db, err = database.OpenInMemory()
    if err != nil {
        return err
    }
    if err := persistence.AutoMigrateModel(db); err != nil {
        return err
    }
} else {
    if cfg.DBAdminDSN != "" {
        // existing admin migration + bootstrap path (unchanged)
    }
    db, err = database.OpenGORM(cfg.DBDSN)
    if err != nil {
        return err
    }
}
sqlDB, err := db.DB()
if err != nil {
    return err
}
defer sqlDB.Close()
```

## Behavior

| Condition | DB used | AutoMigrate | BootstrapSQL |
|---|---|---|---|
| `DBDSN` set | PostgreSQL | Via `DBAdminDSN` (if set) | Via `DBAdminDSN` (if set) |
| `DBDSN` empty | SQLite `:memory:` | Yes, on startup | Skipped (naturally — lives in admin block) |

- The `slog.Warn` line makes in-memory mode visible in logs, preventing silent use in production.
- The SQLite `:memory:` DB is tied to a single connection; closing it destroys all data (correct for ephemeral test use).
- All query filters in `EventRepository.Query()` use basic SQL that SQLite supports.

## Verification

1. **No env vars set:** `go run ./cmd/server` should start without error and log the in-memory warning.
2. **Postgres path unchanged:** Setting `AUDIT_LOG_DB_DSN` should behave exactly as before.
3. **Functional tests:** Run the service in-memory and execute gRPC calls (Save, FindByID, Query with filters) — all should work.
4. **Existing tests unaffected:** `go test ./...` should pass with no changes to test files.
