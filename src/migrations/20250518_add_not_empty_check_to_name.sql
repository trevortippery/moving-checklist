-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks
ADD CONSTRAINT task_name_not_empty CHECK (char_length(trim(name)) > 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tasks
DROP CONSTRAINT task_name_not_empty;
-- +goose StatementEnd