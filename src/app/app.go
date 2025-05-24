package app

import (
	"database/sql"
	"log"
	"os"

	"github.com/trevortippery/moving-checklist/api"
	"github.com/trevortippery/moving-checklist/db"
	"github.com/trevortippery/moving-checklist/migrations"
)

type Application struct {
	Logger      *log.Logger
	TaskHandler *api.TaskHandler
	DB          *sql.DB
}

func NewApplication() (*Application, error) {
	database, err := db.Open()
	if err != nil {
		return nil, err
	}

	err = db.MigrateFS(database, migrations.FS, ".")
	if err != nil {
		return nil, err
	}

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	taskStore := db.NewPostgresTaskStore(database)

	taskHandler := api.NewTaskHandler(taskStore, logger)

	app := &Application{
		Logger:      logger,
		TaskHandler: taskHandler,
		DB:          database,
	}

	return app, nil
}
