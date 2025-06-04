-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tokens (
  token TEXT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  expiry TIMESTAMP WITH TIME ZONE NOT NULL,
  scope TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tokens;
-- +goose StatementEnd
