-- +goose Up
CREATE TABLE posts (
id              UUID        NOT NULL PRIMARY KEY,
created_at      TIMESTAMP   NOT NULL,
updated_at      TIMESTAMP   NOT NULL,
title           TEXT        NOT NULL,
url             TEXT        NOT NULL UNIQUE,
description     TEXT,
published_at    TIMESTAMP,
feed_id         UUID        NOT NULL REFERENCES feeds(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS posts;