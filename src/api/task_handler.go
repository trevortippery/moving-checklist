package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/trevortippery/moving-checklist/db"
	"github.com/trevortippery/moving-checklist/middleware"
	"github.com/trevortippery/moving-checklist/utils"
)

type TaskHandler struct {
	task   db.TaskStore
	logger *log.Logger
}

type TaskRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	IsComplete  bool   `json:"is_complete"`
	DueDate     string `json:"due_date"`
}

type ValidationMode string

const (
	ValidateCreate ValidationMode = "create"
	ValidateUpdate ValidationMode = "update"
)

func NewTaskHandler(taskStore db.TaskStore, logger *log.Logger) *TaskHandler {
	return &TaskHandler{
		task:   taskStore,
		logger: logger,
	}
}

func (th *TaskHandler) HandleCreateTask(w http.ResponseWriter, r *http.Request) {
	const funcName = "HandleCreateTask"

	user := middleware.GetUser(r)
	if user == nil {
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "not authenticated"})
		return
	}

	var input TaskRequest
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		th.logger.Printf("Error in %s: Decoding Request - %v", funcName, err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid request body"})
		return
	}

	validationErrors := validateTaskInput(input, ValidateCreate)
	if len(validationErrors) > 0 {
		th.logger.Printf("Error in %s: Validating input - %+v", funcName, validationErrors)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"errors": validationErrors})
		return
	}

	var dueDate sql.NullTime
	if input.DueDate != "" {
		parsed, parseErr := time.Parse(time.RFC3339, input.DueDate)
		if parseErr != nil {
			utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid due_date format, must be RFC3339"})
			return
		}
		dueDate = sql.NullTime{Time: parsed, Valid: true}
	}

	task := db.Task{
		UserID:      user.ID,
		Name:        input.Name,
		Description: input.Description,
		Category:    input.Category,
		IsComplete:  input.IsComplete,
		DueDate:     dueDate,
	}

	createdTask, err := th.task.CreateTask(r.Context(), &task)
	if err != nil {
		th.logger.Printf("Error in %s: Creating task - %v", funcName, err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "failed to create task"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"task": createdTask})
}

func (th *TaskHandler) HandleDeleteTask(w http.ResponseWriter, r *http.Request) {
	const funcName = "HandleDeleteTask"

	user := middleware.GetUser(r)
	if user == nil {
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "not authenticated"})
		return
	}

	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		th.logger.Printf("%s: Invalid ID - ID parameter is empty", funcName)
		http.NotFound(w, r)
		return
	}

	urlTaskID, err := strconv.ParseInt(taskID, 10, 64)
	if err != nil || urlTaskID <= 0 {
		th.logger.Printf("%s: Invalid ID - %v", funcName, err)
		http.NotFound(w, r)
		return
	}

	err = th.task.DeleteTask(r.Context(), urlTaskID, user.ID)
	if errors.Is(err, db.ErrTaskNotFound) {
		th.logger.Printf("Error in %s: Task not found %d - %v", funcName, urlTaskID, err)
		utils.WriteJSON(w, http.StatusNotFound, utils.Envelope{"error": "task not found"})
		return
	}

	if err != nil {
		th.logger.Printf("Error in %s: Deleting task %d - %v", funcName, urlTaskID, err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "failed to delete task"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (th *TaskHandler) HandleUpdateTask(w http.ResponseWriter, r *http.Request) {
	const funcName = "HandleUpdateTask"

	user := middleware.GetUser(r)
	if user == nil {
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "not authenticated"})
		return
	}

	taskID, err := utils.ReadIDParam(r)
	if err != nil {
		th.logger.Printf("Error in %s: Reading ID from url - %v", funcName, err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid task ID"})
		return
	}

	existingTask, err := th.task.GetTaskByID(r.Context(), taskID, user.ID)
	if errors.Is(err, db.ErrTaskNotFound) {
		th.logger.Printf("Error in %s: Get task by ID - %v", funcName, err)
		utils.WriteJSON(w, http.StatusNotFound, utils.Envelope{"error": "task not found"})
		return
	}

	if err != nil {
		th.logger.Printf("Error in %s: Get task by ID - %v", funcName, err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "internal server error"})
		return
	}

	var updateTaskRequest struct {
		Name        *string       `json:"name"`
		Description *string       `json:"description"`
		Category    *string       `json:"category"`
		IsComplete  *bool         `json:"is_complete"`
		DueDate     *sql.NullTime `json:"due_date"`
	}

	defer r.Body.Close()
	err = json.NewDecoder(r.Body).Decode(&updateTaskRequest)

	if err != nil {
		th.logger.Printf("Error in %s: Decoding update task - %v", funcName, err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid request payload"})
		return
	}

	taskReq := TaskRequest{
		Name:     derefString(updateTaskRequest.Name),
		Category: derefString(updateTaskRequest.Category),
		DueDate:  derefNullTime(updateTaskRequest.DueDate),
	}

	validationErrors := validateTaskInput(taskReq, ValidateUpdate)
	if len(validationErrors) > 0 {
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"errors": validationErrors})
		return
	}

	if updateTaskRequest.Name != nil {
		existingTask.Name = *updateTaskRequest.Name
	}

	if updateTaskRequest.Description != nil {
		existingTask.Description = *updateTaskRequest.Description
	}

	if updateTaskRequest.Category != nil {
		existingTask.Category = *updateTaskRequest.Category
	}

	if updateTaskRequest.IsComplete != nil {
		existingTask.IsComplete = *updateTaskRequest.IsComplete
	}

	if updateTaskRequest.DueDate != nil {
		existingTask.DueDate = *updateTaskRequest.DueDate
	}

	now := time.Now().UTC()
	existingTask.UpdatedAt = sql.NullTime{Time: now, Valid: true}

	err = th.task.UpdateTask(r.Context(), existingTask)

	if errors.Is(err, db.ErrTaskNotFound) {
		th.logger.Printf("Error in %s: Updating task - %v", funcName, err)
		utils.WriteJSON(w, http.StatusNotFound, utils.Envelope{"error": "task not found"})
		return
	}

	if err != nil {
		th.logger.Printf("Error in %s: Updating task - %v", funcName, err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "could not update task"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"task": existingTask})
}

func (th *TaskHandler) HandleGetTaskByID(w http.ResponseWriter, r *http.Request) {
	const funcName = "HandleGetTaskByID"

	user := middleware.GetUser(r)
	if user == nil {
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "not authenticated"})
		return
	}

	taskID, err := utils.ReadIDParam(r)
	if err != nil {
		th.logger.Printf("Error in %s: Reading task ID - %v", funcName, err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid task ID"})
		return
	}

	requestedTask, err := th.task.GetTaskByID(r.Context(), taskID, user.ID)
	if errors.Is(err, db.ErrTaskNotFound) {
		th.logger.Printf("Error in %s: Getting task by ID - %v", funcName, err)
		utils.WriteJSON(w, http.StatusNotFound, utils.Envelope{"error": "task not found"})
		return
	}

	if err != nil {
		th.logger.Printf("Error in %s: Getting task by ID - %v", funcName, err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "could not retrieve task"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"task": requestedTask})
}

func validateTaskInput(input TaskRequest, mode ValidationMode) map[string]string {
	errors := make(map[string]string)

	if mode == ValidateCreate {
		if strings.TrimSpace(input.Name) == "" {
			errors["name"] = "name is required"
		} else if len(input.Name) > 100 {
			errors["name"] = "name must be less than 100 characters"
		}
	}

	if len(input.Category) > 50 {
		errors["category"] = "category must be less than 50 characters"
	}

	if input.DueDate != "" {
		if _, err := time.Parse(time.RFC3339, input.DueDate); err != nil {
			errors["due_date"] = "due_date must be in RFC3339 format (e.g., 2025-05-17T15:04:05Z)"
		}
	}

	return errors
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefNullTime(nt *sql.NullTime) string {
	if nt == nil || !nt.Valid {
		return ""
	}
	return nt.Time.Format(time.RFC3339)
}
