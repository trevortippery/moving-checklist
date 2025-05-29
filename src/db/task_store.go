package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Task struct {
	ID          int          `json:"id"`
	UserID      int          `json:"user_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Category    string       `json:"category"`
	IsComplete  bool         `json:"is_complete"`
	DueDate     sql.NullTime `json:"due_date"`
	CreatedAt   sql.NullTime `json:"created_at"`
	UpdatedAt   sql.NullTime `json:"updated_at"`
}

type PostgresTaskStore struct {
	db *sql.DB
}

func NewPostgresTaskStore(db *sql.DB) *PostgresTaskStore {
	return &PostgresTaskStore{db: db}
}

type TaskStore interface {
	CreateTask(ctx context.Context, task *Task) (*Task, error)
	DeleteTask(ctx context.Context, id int64) error
	UpdateTask(ctx context.Context, task *Task) error
	GetTaskByID(ctx context.Context, id int64) (*Task, error)
}

func (pg *PostgresTaskStore) CreateTask(ctx context.Context, task *Task) (*Task, error) {
	transaction, err := pg.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer transaction.Rollback()

	query := `
	INSERT INTO tasks (name, description, category, is_complete, due_date, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id
	`

	err = transaction.QueryRowContext(ctx, query,
		task.Name,
		task.Description,
		task.Category,
		task.IsComplete,
		task.DueDate,
		task.CreatedAt,
		task.UpdatedAt,
	).Scan(&task.ID)

	if err != nil {
		return nil, err
	}

	err = transaction.Commit()
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (pg *PostgresTaskStore) DeleteTask(ctx context.Context, id int64) error {
	query := ` DELETE FROM tasks WHERE id = $1`
	result, err := pg.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no task with id %d: %w", id, sql.ErrNoRows)
	}

	return nil
}

func (pg *PostgresTaskStore) UpdateTask(ctx context.Context, task *Task) error {
	if task == nil {
		return errors.New("cannot update nil task")
	}

	transaction, err := pg.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer transaction.Rollback()

	query := `
	UPDATE tasks
	SET name = $1, description = $2, category = $3, is_complete = $4, due_date = $5, updated_at = $6
	WHERE id = $7
	`

	result, err := transaction.ExecContext(ctx, query,
		task.Name,
		task.Description,
		task.Category,
		task.IsComplete,
		task.DueDate,
		task.UpdatedAt,
		task.ID)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	err = transaction.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (pg *PostgresTaskStore) GetTaskByID(ctx context.Context, id int64) (*Task, error) {
	task := &Task{}

	query := `
	SELECT id, name, description, category, is_complete, due_date, created_at, updated_at
	FROM tasks
	WHERE id = $1
	`

	err := pg.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.Name,
		&task.Description,
		&task.Category,
		&task.IsComplete,
		&task.DueDate,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return task, nil
}
