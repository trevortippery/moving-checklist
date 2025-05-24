package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/trevortippery/moving-checklist/db"
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
	var input TaskRequest
	var funcName = "HandleCreateTask"

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		th.logger.Printf("Error in %s: Decoding Create Task - %v", funcName, err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid request sent"})
		return
	}

	//area for valdiating fields
	validationErrors := validateTaskInput(input, ValidateCreate)
	if len(validationErrors) > 0 {
		th.logger.Printf("Error in %s: Validating input - %+v", funcName, validationErrors)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"errors": validationErrors})
		return
	}

	// area for checking correct user

	var dueDate sql.NullTime
	if input.DueDate != "" {
		parsed, _ := time.Parse(time.RFC3339, input.DueDate)
		dueDate = sql.NullTime{Time: parsed, Valid: true}
	}

	now := time.Now().UTC()

	task := db.Task{
		Name:        input.Name,
		Description: input.Description,
		Category:    input.Category,
		IsComplete:  input.IsComplete,
		DueDate:     dueDate,
		CreatedAt:   sql.NullTime{Time: now, Valid: true},
		UpdatedAt:   sql.NullTime{Time: now, Valid: true},
	}

	createdTask, err := th.task.CreateTask(r.Context(), &task)
	if err != nil {
		th.logger.Printf("Error in %s: Creating task - %v", funcName, err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "failed to create task"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"task": createdTask})
}

func (th *TaskHandler) HandleDeleteTaskByID(w http.ResponseWriter, r *http.Request) {
	funcName := "HandleDeleteTaskByID"

	// Log the full URL path
	fmt.Println("Request URL:", r.URL.Path) // Log the full URL path

	// Directly extract the ID using chi.URLParam
	taskID := chi.URLParam(r, "id")
	fmt.Println("Extracted Task ID:", taskID)

	if taskID == "" {
		th.logger.Printf("%s: Invalid ID - ID parameter is empty", funcName)
		http.NotFound(w, r)
		return
	}

	// Convert the taskID to an integer
	urlTaskID, err := strconv.ParseInt(taskID, 10, 64)
	if err != nil || urlTaskID <= 0 {
		th.logger.Printf("%s: Invalid ID - %v", funcName, err)
		http.NotFound(w, r)
		return
	}

	th.logger.Printf("%s: Attempting to delete task with ID: %d", funcName, urlTaskID)

	// Call delete logic
	err = th.task.DeleteTask(r.Context(), urlTaskID)
	if errors.Is(err, sql.ErrNoRows) {
		th.logger.Printf("Error in %s: Task not found %d - %v", funcName, urlTaskID, err)
		http.Error(w, "Error: task not found", http.StatusNotFound)
		return
	}

	if err != nil {
		th.logger.Printf("Error in %s: Deleting task %d - %v", funcName, urlTaskID, err)
		http.Error(w, "Error: deleting task", http.StatusInternalServerError)
		return
	}

	th.logger.Printf("%s: Successfully deleted task %d", funcName, urlTaskID)
	w.WriteHeader(http.StatusNoContent)
}

func (th *TaskHandler) HandleUpdateTaskByID(w http.ResponseWriter, r *http.Request) {
	funcName := "HandleUpdateTaskByID"
	taskID, err := utils.ReadIDParam(r)

	if err != nil {
		th.logger.Printf("Error in %s: Reading ID from url - %v", funcName, err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid request sent"})
		return
	}

	existingTask, err := th.task.GetTaskByID(r.Context(), taskID)
	if err != nil {
		th.logger.Printf("Error in %s: Get task by ID - %v", funcName, err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "internal server error"})
		return
	}

	if existingTask == nil {
		utils.WriteJSON(w, http.StatusNotFound, utils.Envelope{"error": "task not found"})
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

	// TODO: Add user authentication checking if user is authroized to update this task

	err = th.task.UpdateTask(r.Context(), existingTask)
	if err != nil {
		th.logger.Printf("Error in %s: Updating task - %v", funcName, err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error": "could not update task"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"task": existingTask})
}

func (th *TaskHandler) HandleGetTaskByID(w http.ResponseWriter, r *http.Request) {
	funcName := "HandleGetTaskByID"
	taskID, err := utils.ReadIDParam(r)

	if err != nil {
		th.logger.Printf("Error in %s: Reading ID from URL - %v", funcName, err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error": "invalid request sent"})
		return
	}

	requestedTask, err := th.task.GetTaskByID(r.Context(), taskID)
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
