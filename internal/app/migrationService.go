package app

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gomigrator/internal/domain"
	"html/template"
	"io/fs"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

const MIGRATION_TEMPLATE = `package app

import "database/sql"

func init() {
	migrator.AddMigration(&MigrationSql{
		Version: "{{.Version}}",
		Up:      mig_{{.Version}}_{{.Name}}_up,
		Down:    mig_{{.Version}}_{{.Name}}_down,
	})
}

func mig_{{.Version}}_{{.Name}}_up(tx *sql.Tx) error {
	return nil
}

func mig_{{.Version}}_{{.Name}}_down(tx *sql.Tx) error {
	return nil
}`


type MigrationService struct {
	repo           domain.SchemaRepository
	migrationDir   string
	codeGenService CodeGenServiceImpl
	isSql bool
}

func NewMigrationService(repository domain.SchemaRepository, migrationDir string, isSql bool ) *MigrationService {
	service := new(MigrationService)
	service.repo = repository
	service.migrationDir = migrationDir
	service.codeGenService = CodeGenServiceImpl{}
	service.isSql = isSql
	service.Init()

	return service
}

func (s *MigrationService) Init() error {
	query := "CREATE TABLE IF NOT EXISTS schema_migrations (id INT NOT NULL PRIMARY KEY, version INT NOT NULL); "
	err := s.repo.Exec(query)
	if err != nil {
		log.Info(errors.Wrap(err, "create schema_migration table "))
		return err
	}

	migrationSchema, err := s.repo.Find()
	if err != nil {
		log.Info(errors.Wrap(err, "create schema_migration table "))
		return err
	}

	version := migrationSchema.Version
	if s.isSql {
		err = s.Up(migrationSchema)
		if err != nil {
			log.Info(err)
			return err
		}
	} else {

	}
	return nil
}

func (s *MigrationService) Create(name string) error {
	version := time.Now().Format("20060102150405")
	in := struct {
		Version string
		Name    string
	}{
		Version: version,
		Name:    name,
	}
	var out bytes.Buffer
	tmpl, err := template.New(name).Parse(MIGRATION_TEMPLATE)
	if err != nil {
		return errors.New("unable to parse template: " + err.Error())
	}
	//tmpl := template.Must(template.ParseGlob(MIGRATION_TEMPLATE))
	err = tmpl.Execute(&out, in)
	if err != nil {
		return errors.New("unable to execute template: " + err.Error())
	}
	file, err := os.Create(fmt.Sprintf(s.migrationDir+"/%s_%s.go", version, name))
	if err != nil {
		return errors.New("unable to create migration file: " + err.Error())
	}
	defer file.Close()

	if _, err = file.WriteString(out.String()); err != nil {
		return errors.New("Unable to write to migration file:" + err.Error())
	}
	fmt.Println("Generated new migration files...", file.Name())
	return nil
}

func (s *MigrationService) Generate(pathToModels []string, outputPath string) error {
	for _, model := range pathToModels {

		s.codeGenService.Generate(model, outputPath)
	}
	return nil
}

func (s *MigrationService) Up(version domain.SchemaMigration) error {
	lastMigrationVersion, err := s.repo.Find()
	if err != nil {
		if err != domain.ErrSchemaTableEmpty {
			log.Error("Failed isUpAction last migration version")
		} else {
		}
	}
	files, err := s.getFiles(err)
	if err != nil {
		return err
	}

	for _, file := range files {
		fmt.Println(file)
		var migrationFileVersion int
		migrationFileVersion, err = s.getMigrationFileVersion(file)
		if err != nil {
			return err
		}
		contains := s.isUpAction(file)
		if migrationFileVersion > lastMigrationVersion.Version && contains {
			err = s.execMigration(file, err, migrationFileVersion)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *MigrationService) isUpAction(file fs.FileInfo) bool {
	i := len(file.Name())
	s2 := file.Name()[i-9:]
	contains := strings.Contains(s2, ".up.sql")
	return contains
}

func (s *MigrationService) execMigration(file fs.FileInfo, err error, migrationFileVersion int) error {
	filePath := s.migrationDir + string(os.PathSeparator) + file.Name()
	sql, err := s.getFileContent(filePath)
	if err != nil {
		return err
	}
	fmt.Println(sql)
	err = s.repo.Exec(sql)
	if err != nil {
		log.Error(err)
	}

	log.Info("version %d", migrationFileVersion)
	err = s.repo.Update(domain.SchemaMigration{Version: migrationFileVersion})
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (s *MigrationService) getFileContent(filePath string) (string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	sql := string(content)
	return sql, nil
}

func (s *MigrationService) getMigrationFileVersion(file fs.FileInfo) (int, error) {
	split := strings.Split(file.Name(), "_")
	return strconv.Atoi(split[0])
}

func (s *MigrationService) getFiles(err error) ([]fs.FileInfo, error) {
	files, err := ioutil.ReadDir(s.migrationDir)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	fmt.Println(len(files))
	return files, nil
}

func (s *MigrationService) Down(version domain.SchemaMigration) error {
	lastMigration, err := s.repo.Find()
	if err != nil {
		if err == domain.ErrSchemaTableEmpty {
			return nil
		}
		return err
	}
	var file fs.FileInfo
	file, err = s.getNeededMigration(err, lastMigration)
	if err != nil {
		return err
	}
	if file == nil {
		return errors.New("nothing to undo")
	}
	err = s.execMigration(file, err, lastMigration.Version-1)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (s *MigrationService) getNeededMigration(err error, lastMigration domain.SchemaMigration) (fs.FileInfo, error) {
	files, err := s.getFiles(err)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		var migrationFileVersion int
		migrationFileVersion, err = s.getMigrationFileVersion(file)
		if err != nil {
			return nil, err
		}

		if migrationFileVersion == lastMigration.Version && !s.isUpAction(file) {
			return file, nil
		}
	}
	return nil, err
}

func (s *MigrationService) Redo() error {
	return nil
}

func (s *MigrationService) Undo() error {
	return nil
}

func (s *MigrationService) GetVersion() (int, error) {
	return 0, nil
}
