// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: permissions.sql

package database

import (
	"context"

	"github.com/google/uuid"
)

const getAllPermissionsForUser = `-- name: GetAllPermissionsForUser :many
SELECT permissions.code
FROM permissions
INNER JOIN users_permissions ON users_permissions.permissions_id = permissions.id
INNER JOIN users ON users_permissions.user_id = users.id
WHERE users.id = $1
`

func (q *Queries) GetAllPermissionsForUser(ctx context.Context, id uuid.UUID) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, getAllPermissionsForUser, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		items = append(items, code)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
