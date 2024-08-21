-- +goose Up
CREATE TABLE tokens (
hash        bytea           PRIMARY KEY,
user_id     UUID            NOT NULL,
expiry      TIMESTAMP(0)    with time zone NOT NULL,
scope       text            NOT NULL,
FOREIGN KEY (user_id)       REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS tokens;