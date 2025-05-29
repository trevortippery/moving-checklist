-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks
ADD COLUMN user_id BIGINT NOT NULL,
ADD CONSTRAINT fk_user
    FOREIGN KEY (user_id)
    REFERENCES users(id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks
DROP CONSTRAINT IF EXISTS fk_user,
DROP COLUMN IF EXISTS user_id;
-- +goose StatementEnd
