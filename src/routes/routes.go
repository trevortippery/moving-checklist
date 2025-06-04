package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/trevortippery/moving-checklist/app"
	"github.com/trevortippery/moving-checklist/middleware"
)

func SetupRoutes(app *app.Application) *chi.Mux {
	r := chi.NewRouter()

	// Tasks routes - require auth
	r.Route("/tasks", func(r chi.Router) {
		r.Use(app.Middleware.Authenticate)
		r.Use(middleware.RequireUser)

		r.Post("/", app.TaskHandler.HandleCreateTask)
		r.Put("/{id}", app.TaskHandler.HandleUpdateTask)
		r.Delete("/{id}", app.TaskHandler.HandleDeleteTask)
		r.Get("/{id}", app.TaskHandler.HandleGetTaskByID)
	})

	// User registration is public
	r.Post("/users", app.UserHandler.HandleRegisterUser)

	// User routes - require auth
	r.Route("/users", func(r chi.Router) {
		r.Use(app.Middleware.Authenticate)
		r.Use(middleware.RequireUser)

		r.Delete("/me", app.UserHandler.HandleDeleteUser)
		r.Put("/me", app.UserHandler.HandleUpdateUser)
	})

	return r
}
