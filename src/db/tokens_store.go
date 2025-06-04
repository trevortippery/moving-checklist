package db

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"time"
)

type Token struct {
	UserID int64
	Token  string
	Expiry time.Time
	Scope  string
}

type PostgresTokenStore struct {
	db *sql.DB
}

type TokenStore interface {
	GenerateToken(ctx context.Context, userID int64, ttl time.Duration, scope string) (string, error)
}

func NewPostgresTokenStore(db *sql.DB) *PostgresTokenStore {
	return &PostgresTokenStore{db: db}
}

func (ts *PostgresTokenStore) GenerateToken(ctx context.Context, userID int64, ttl time.Duration, scope string) (string, error) {

	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}

	rawToken := base64.URLEncoding.EncodeToString(tokenBytes)

	hash := sha256.Sum256([]byte(rawToken))
	hashedToken := base64.URLEncoding.EncodeToString(hash[:])

	expiry := time.Now().Add(ttl)

	query := `
		INSERT INTO tokens (token, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4)
	`

	_, err = ts.db.ExecContext(ctx, query, hashedToken, userID, expiry, scope)
	if err != nil {
		return "", err
	}

	return rawToken, nil
}

// (Optional) You can add methods like DeleteToken, GetToken, etc.
