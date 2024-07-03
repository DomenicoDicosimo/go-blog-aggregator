-- +goose Up
CREATE TABLE users (
id UUID NOT NULL,
created_at TIMESTAMP NOT NULL,
updated_at TIMESTAMP NOT NULL,
name TEXT NOT NULL,
Primary Key(id)
);
-- +goose Down
DROP TABLE users;