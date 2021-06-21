package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gomigrator/internal/app"
	"gomigrator/internal/domain"
	"gomigrator/internal/infrastructure/postgres"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}
	logger := log.New()
	logger.SetFormatter(&log.JSONFormatter{})
	logger.Print("Starting the service...")
	db := initDB(logger)
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()

	migrationDir := flag.String("migrationDir", "C:\\Users\\mikha\\go\\src\\gomigrator\\bin\\gomigrator\\migration", "db host")

	repo := postgres.NewSchemaRepository(db)
	migrationService := app.NewMigrationService(repo, *migrationDir)
	command := os.Args[1]

	switch command {
	case "up":
		err := migrationService.Up(domain.SchemaMigration{})
		if err != nil {
			fmt.Println(err.Error())
		}
		break
	case "down":
		err := migrationService.Down(domain.SchemaMigration{})
		if err != nil {
			fmt.Println(err.Error())
		}
		break
	case "create":
		if len(os.Args) < 3 {
			printHelp()
			break
		}
		err := migrationService.Create(os.Args[2])
		if err != nil {
			fmt.Println(err.Error())
		}
		break
	case "generate":
		if len(os.Args) < 4 {
			printHelp()
			break
		}
		outputPath := os.Args[2]
		models := os.Args[3:]
		err := migrationService.Generate(models, outputPath)
		if err != nil {
			fmt.Println(err.Error())
		}
		break
	default:
		fmt.Println("Unknown command")
		break
	}

}

func printHelp() {
	fmt.Println("usage:  gomigrator <up|down|redo|status|dbversion>|<create> <migration_version> <domain>")
}

func initDB(logger *log.Logger) *sql.DB {
	dbHost := flag.String("host", "localhost", "db host")
	dbPort := flag.Int("port", 5432, "postgresql dsn")
	dbName := flag.String("dbName", "migrationtest", "db host")
	dbUser := flag.String("dbUser", "postgres", "db host")
	dbPassword := flag.String("dbPassword", "postgres", "db host")
	postgresSource := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		*dbHost, *dbPort, *dbUser, *dbPassword, *dbName)
	for {
		db, err := sql.Open("postgres", postgresSource)
		if err != nil {
			logger.Info(errors.Wrap(err, "can't open connection to "+postgresSource))
			time.Sleep(time.Second)
			continue
		}

		err = db.Ping()
		if err != nil {
			logger.Info(errors.Wrap(err, "can't ping to "+postgresSource))
			time.Sleep(time.Second)
			continue
		}
		return db
	}
}
