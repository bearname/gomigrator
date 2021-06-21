package postgres

import (
	"database/sql"
	"gomigrator/internal/domain"
)

type SchemaRepository struct {
	db *sql.DB
}

func NewSchemaRepository(db *sql.DB) domain.SchemaRepository {
	return &SchemaRepository{db: db}
}

func (repo *SchemaRepository) Exec(sqlQuery string) error {
	_, err := repo.db.Exec(sqlQuery)
	return err
}

func (repo *SchemaRepository) Find() (domain.SchemaMigration, error) {
	sqlStatement := `SELECT version FROM schema_migrations LIMIT 1;`
	var version domain.SchemaMigration

	row := repo.db.QueryRow(sqlStatement)
	err := row.Scan(&version.Version)
	if err == nil {
		return version, nil
	}
	switch err {
	case sql.ErrNoRows:
		{
			version.Version = -1
			return version, domain.ErrSchemaTableEmpty
		}
	default:
		return version, err
	}
}

func (repo *SchemaRepository) Update(schemaMigration domain.SchemaMigration) error {
	migration, err := repo.Find()
	var sqlStatement string

	if err != nil {
		if err == domain.ErrSchemaTableEmpty {
			sqlStatement = `INSERT INTO schema_migrations (id, version) VALUES ($1, $2); `
		}
	} else {
		if migration.Version == schemaMigration.Version {
			return nil
		}
		sqlStatement = `UPDATE schema_migrations SET version = $1 WHERE id=$2;`
	}
	_, err = repo.db.Exec(sqlStatement, schemaMigration.Version, 1)
	if err != nil {
		return err
	}

	return nil
}