package domain

type MigrationService interface {
	Init() error
	Create(name string) error
	Generate(pathToModels []string, outputPath string) error
	Up(version SchemaMigration) error
	Down(version SchemaMigration) error
	Redo() error
	Undo() error
	GetVersion() (int, error)
}
