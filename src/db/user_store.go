package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PostgresUserStore struct {
	db *sql.DB
}

func NewPostgresUserStore(db *sql.DB) *PostgresUserStore {
	return &PostgresUserStore{db: db}
}

type UserStore interface {
	RegisterUser(ctx context.Context, user *User) (*User, error)
	DeleteUser(ctx context.Context, id int64) error
	UpdateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id int64) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CheckEmailExists(ctx context.Context, email string) (bool, error)
	GetUserByToken(ctx context.Context, token string, scope string) (*User, error)
}

func (pg *PostgresUserStore) RegisterUser(ctx context.Context, user *User) (*User, error) {
	query := `
	INSERT INTO users (username, email, password_hash)
	VALUES ($1, $2, $3)
	returning id, created_at, updated_at
	`

	err := pg.db.QueryRowContext(ctx, query, user.Username, user.Email, user.PasswordHash).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (pg *PostgresUserStore) DeleteUser(ctx context.Context, id int64) error {
	query := `
	DELETE FROM users
	WHERE id = $1
	`

	result, err := pg.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no user with id %d: %w", id, sql.ErrNoRows)
	}

	return nil
}

func (pg *PostgresUserStore) UpdateUser(ctx context.Context, user *User) error {
	if user == nil {
		return errors.New("cannot update nil user")
	}

	transaction, err := pg.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer transaction.Rollback()

	query := `
	UPDATE users
	SET username = $1, email = $2, password_hash = $3, updated_at = CURRENT_TIMESTAMP
	WHERE id = $4
	`

	result, err := transaction.ExecContext(ctx, query,
		user.Username,
		user.Email,
		user.PasswordHash,
		user.ID,
	)

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

func (pg *PostgresUserStore) GetUserByID(ctx context.Context, id int64) (*User, error) {
	user := &User{}

	query := `
	SELECT id, username, email, created_at, updated_at
	FROM users
	WHERE id = $1
	`

	err := pg.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (pg *PostgresUserStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}

	query := `
	SELECT id, username, email, password_hash, created_at, updated_at
	FROM users
	WHERE email = $1
	`

	err := pg.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (pg *PostgresUserStore) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	query := "SELECT 1 from users WHERE email = $1 LIMIT 1"

	var exists int
	err := pg.db.QueryRowContext(ctx, query, email).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (pg *PostgresUserStore) GetUserByToken(ctx context.Context, token string, scope string) (*User, error) {

	hash := sha256.Sum256([]byte(token))
	hashedToken := base64.URLEncoding.EncodeToString(hash[:])

	var user User
	query := `
			SELECT u.id, u.username, u.email, u.password_hash, u.created_at, u.updated_at
			FROM users u
			INNER JOIN tokens t ON u.id = t.user_id
			WHERE t.token = $1 AND t.scope = $2 AND t.expiry > CURRENT_TIMESTAMP
			LIMIT 1;
	`
	err := pg.db.QueryRowContext(ctx, query, hashedToken, scope).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // token not found or expired
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}
