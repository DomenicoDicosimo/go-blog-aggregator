-- name: InsertUser :one
INSERT INTO users (id, created_at, updated_at, name, email, password_hash, activated, version)
VALUES (
  $1, -- id
  $2, -- created_at
  $3, -- updated_at
  $4, -- name
  $5, -- email
  $6, -- password_hash
  $7, -- activated
  1  -- version (default to 1 for new users)
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT id, created_at, updated_at, name, email, password_hash, activated, version
FROM users
WHERE email = $1;

-- name: UpdateUser :one
UPDATE users
SET name = $2,
    email = $3,
    password_hash = $4,
    activated = $5,
    updated_at = $6,
    version = version + 1
WHERE id = $1 AND version = $7
RETURNING version;