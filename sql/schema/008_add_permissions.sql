-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS permissions (
    id          UUID    PRIMARY KEY     DEFAULT uuid_generate_v4(),
    code        text    NOT NULL        UNIQUE
);

CREATE TABLE IF NOT EXISTS users_permissions (
    user_id         UUID    NOT NULL REFERENCES users(id)       ON DELETE CASCADE,
    permissions_id  UUID    NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY     (user_id, permissions_id)
);

INSERT INTO permissions (code)
VALUES
    ('feeds:read'),
    ('feeds:write'),
    ('feed_follows:write'),
    ('feed_follows:read'),
    ('posts:read');

-- +goose Down
DROP TABLE      IF EXISTS   users_permissions;
DROP TABLE      IF EXISTS   permissions;
DROP EXTENSION  IF EXISTS   "uuid-ossp";