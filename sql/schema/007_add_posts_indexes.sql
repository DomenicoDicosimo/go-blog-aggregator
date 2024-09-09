-- +goose Up
CREATE INDEX IF NOT EXISTS movies_title_idx ON posts USING GIN (to_tsvector('simple', title));

-- +goose Down
DROP INDEX IF EXISTS movies_title_idx;