package app

import (
	"database/sql"
	"log"
	"os"

	"github.com/trevortippery/moving-checklist/api"
	"github.com/trevortippery/moving-checklist/db"
	"github.com/trevortippery/moving-checklist/middleware"
	"github.com/trevortippery/moving-checklist/migrations"
)

type Application struct {
	Logger      *log.Logger
	TaskHandler *api.TaskHandler
	UserHandler *api.UserHandler
	Middleware  *middleware.AuthMiddleware
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
	userStore := db.NewPostgresUserStore(database)
	tokenStore := db.NewPostgresTokenStore(database)

	taskHandler := api.NewTaskHandler(taskStore, logger)
	userHandler := api.NewUserHandler(userStore, tokenStore, logger)
	middlewareHandler := &middleware.AuthMiddleware{UserStore: userStore}

	app := &Application{
		Logger:      logger,
		TaskHandler: taskHandler,
		UserHandler: userHandler,
		Middleware:  middlewareHandler,
		DB:          database,
	}

	return app, nil
}
