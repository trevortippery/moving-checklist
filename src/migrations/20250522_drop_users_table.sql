-- +goose Up
DROP TABLE IF EXISTS users;

-- +goose Down
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL
);
