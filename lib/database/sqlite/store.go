// Package sqlite provides opening a GORM SQLite connection only.
// Table models and CRUD belong to the plugin that uses this DB.
package sqlite

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Open opens a SQLite DB at dbPath with GORM. Caller is responsible for AutoMigrate and all CRUD.
func Open(dbPath string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
}
