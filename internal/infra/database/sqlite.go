package database

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// InMemorySQLiteDSN is used when AUDIT_LOG_DB_DSN is unset. Shared cache ensures every
// connection in the pool sees the same DB (plain :memory: is per-connection, so AutoMigrate
// on one connection left others without tables).
const InMemorySQLiteDSN = "file:audit?mode=memory&cache=shared"

// OpenInMemory opens a SQLite in-memory GORM connection.
// Uses modernc.org/sqlite (pure Go) so the binary works with CGO_ENABLED=0 (e.g. static Docker images).
// The database is ephemeral — data is lost when all connections to it are closed.
func OpenInMemory() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(InMemorySQLiteDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("sqlite open in-memory: %w", err)
	}
	return db, nil
}
