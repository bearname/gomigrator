package domain

import (
	"database/sql"
	"errors"
)

var (
	ErrSchemaTableEmpty = errors.New("schema_migrations table is empty")
)

type SchemaMigration struct {
	Version int
}

type MigrationSql struct {
	Name string
	OutputPath string
}

type MigrationGo struct {
	Version string
	Up      func(*sql.Tx) error
	Down    func(*sql.Tx) error

	done bool
}