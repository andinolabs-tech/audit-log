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
