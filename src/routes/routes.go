package routes

import (
	"github.com/go-chi/chi/v5"
	"github.com/trevortippery/moving-checklist/app"
)

func SetupRoutes(app *app.Application) *chi.Mux {
	r := chi.NewRouter()

	r.Route("/tasks", func(r chi.Router) {
		r.Post("/", app.TaskHandler.HandleCreateTask)
		r.Put("/{id}", app.TaskHandler.HandleUpdateTaskByID)
		r.Delete("/{id}", app.TaskHandler.HandleDeleteTaskByID) // DELETE /tasks/{id}
		r.Get("/{id}", app.TaskHandler.HandleGetTaskByID)       // GET /tasks/{id}
		// r.Get("/", app.TaskHandler.HandleListTasks)             // GET /tasks
	})
	return r
}
