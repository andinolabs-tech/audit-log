package database

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// OpenInMemory opens a SQLite in-memory GORM connection.
// Uses modernc.org/sqlite (pure Go) so the binary works with CGO_ENABLED=0 (e.g. static Docker images).
// The database is ephemeral — data is lost when the connection is closed.
func OpenInMemory() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("sqlite open in-memory: %w", err)
	}
	return db, nil
}
