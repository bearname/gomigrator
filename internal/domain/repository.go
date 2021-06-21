package domain


type SchemaRepository interface {
	Exec(sqlQuery string) error
	Find() ( SchemaMigration, error)
	Update(schemaMigration SchemaMigration) error
}