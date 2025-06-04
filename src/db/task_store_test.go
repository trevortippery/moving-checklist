package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type taskTestCase struct {
	name    string
	task    *Task
	wantErr bool
}

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("pgx", "host=localhost user=postgres password=postgres dbname=postgres port=5433 sslmode=disable")
	if err != nil {
		t.Fatalf("Opening test db: %v", err)
	}

	fmt.Println("Running migrations...")
	err = Migrate(db, "../migrations")
	if err != nil {
		t.Fatalf("Migrating test db error: %v", err)
	}
	fmt.Println("Migrations finished.")

	_, err = db.Exec(`TRUNCATE tasks, users CASCADE`)
	if err != nil {
		t.Fatalf("Truncating tasks and users tables: %v", err)
	}

	return db
}

func TestCreateTask(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := createTestUser(t, db)
	store := NewPostgresTaskStore(db)
	ctx := context.Background()

	tests := []taskTestCase{
		{
			name:    "Valid task",
			task:    validTask("Change of address", user.ID),
			wantErr: false,
		},
		{
			name: "Invalid task",
			task: &Task{
				Name:        "",
				Description: "Missing name field",
				Category:    "general",
				IsComplete:  false,
				DueDate: sql.NullTime{
					Time:  time.Now().Add(48 * time.Hour),
					Valid: true,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createdTask, err := store.CreateTask(ctx, tt.task)

			if tt.wantErr {
				assert.Error(t, err, "expected an error for test case: %s", tt.name)
				assert.Nil(t, createdTask)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, createdTask)
			assert.NotZero(t, createdTask.ID)
			assert.Equal(t, tt.task.Name, createdTask.Name)
			assert.Equal(t, tt.task.Description, createdTask.Description)
			assert.Equal(t, tt.task.Category, createdTask.Category)
			assert.Equal(t, tt.task.IsComplete, createdTask.IsComplete)

			if tt.task.DueDate.Valid {
				assert.True(t, createdTask.DueDate.Valid)
				assert.WithinDuration(t, tt.task.DueDate.Time, createdTask.DueDate.Time, time.Second)
			} else {
				assert.False(t, createdTask.DueDate.Valid)
			}
		})
	}

}

func TestDeleteTask(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresTaskStore(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		prepare func(t *testing.T, db *sql.DB) (taskID int64, userID int)
		wantErr bool
	}{
		{
			name: "Deleting existing task",
			prepare: func(t *testing.T, db *sql.DB) (int64, int) {
				user := createTestUser(t, db)
				task := validTask("Delete me", user.ID)
				createdTask, err := store.CreateTask(ctx, task)
				require.NoError(t, err)
				require.NotNil(t, createdTask)

				return int64(createdTask.ID), user.ID
			},
			wantErr: false,
		},
		{
			name: "Deleting non-existing task",
			prepare: func(t *testing.T, db *sql.DB) (int64, int) {
				user := createTestUser(t, db)
				return 999999, user.ID // bogus task ID, valid user
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskID, userID := tt.prepare(t, db)

			fmt.Printf("Deleting task with ID=%d, UserID=%d\n", taskID, userID)
			err := store.DeleteTask(ctx, taskID, userID)

			if tt.wantErr {
				require.Error(t, err)
				require.True(t, errors.Is(err, ErrTaskNotFound), "expected ErrTaskNotFound, got: %v", err)
			} else {
				require.NoError(t, err)
				task, err := store.GetTaskByID(ctx, taskID, userID)
				require.NoError(t, err)
				require.Nil(t, task)
			}
		})
	}
}

func TestUpdateTask(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := createTestUser(t, db)
	store := NewPostgresTaskStore(db)
	ctx := context.Background()

	tests := []taskTestCase{
		{
			name:    "Successfully updating an existing task",
			task:    validTask("Update me", user.ID),
			wantErr: false,
		},
		{
			name:    "Updating non-existing task (invalid ID)",
			task:    &Task{ID: 99999, Name: "Ghost Task", Description: "Doesn't exist"},
			wantErr: true,
		},
		{
			name:    "Updating nil task",
			task:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			if tt.task == nil {
				err = store.UpdateTask(ctx, nil)
				require.Error(t, err)
				return
			}

			if tt.task.ID == 0 {
				createdTask, err := store.CreateTask(ctx, tt.task)
				require.NoError(t, err)
				require.NotNil(t, createdTask)
				tt.task.ID = createdTask.ID
			}

			tt.task.Name = "Updated Task Name"
			tt.task.Description = "Updated Description"

			err = store.UpdateTask(ctx, tt.task)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Fetch task to confirm update
				updatedTask, err := store.GetTaskByID(ctx, int64(tt.task.ID), user.ID)
				require.NoError(t, err)
				require.Equal(t, "Updated Task Name", updatedTask.Name)
				require.Equal(t, "Updated Description", updatedTask.Description)
			}
		})
	}
}

func TestGetTaskByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user := createTestUser(t, db)
	store := NewPostgresTaskStore(db)
	ctx := context.Background()

	tests := []taskTestCase{
		{
			name:    "Sucessfully getting task by ID",
			task:    validTask("Fetch this task", user.ID),
			wantErr: false,
		},
		{
			name:    "Getting non-existing task (invalid ID)",
			task:    &Task{Name: "Non-existing task", ID: -1},
			wantErr: false,
		},
		{
			name:    "Getting task with zero ID",
			task:    &Task{Name: "Task with zero ID", ID: 0},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var taskID int64

			if tt.task != nil && tt.task.ID == 0 {
				createdTask, err := store.CreateTask(ctx, tt.task)
				require.NoError(t, err)
				require.NotNil(t, createdTask)
				taskID = int64(createdTask.ID)
			} else if tt.task != nil {
				taskID = int64(tt.task.ID)
			} else {
				t.Skip("nil task not supported with current implementation")
			}

			result, err := store.GetTaskByID(ctx, taskID, user.ID)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.task.ID <= 0 {
					require.Nil(t, result)
				} else {
					require.NotNil(t, result)
					require.Equal(t, tt.task.Name, result.Name)
				}
			}
		})
	}
}

func validTask(name string, userID int) *Task {
	return &Task{
		UserID:      userID,
		Name:        name,
		Description: "some description",
		Category:    "work",
		IsComplete:  false,
		DueDate: sql.NullTime{
			Time:  time.Now().Add(48 * time.Hour),
			Valid: true,
		},
	}
}

func createTestUser(t *testing.T, db *sql.DB) *User {
	userStore := NewPostgresUserStore(db)
	user := &User{
		Username:     fmt.Sprintf("testuser_%d", time.Now().UnixNano()),
		Email:        fmt.Sprintf("user%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hashed-password",
	}

	createdUser, err := userStore.RegisterUser(context.Background(), user)
	require.NoError(t, err)
	require.NotNil(t, createdUser)

	return createdUser
}
