-- +goose Up
-- +goose StatementBegin
ALTER TABLE tokens
  ALTER COLUMN token TYPE CHAR(44),
  ALTER COLUMN token SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tokens_user_scope ON tokens(user_id, scope);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Revert to previous state (optional: adjust based on your actual original schema)
ALTER TABLE tokens
  ALTER COLUMN token TYPE TEXT;

-- +goose StatementEnd