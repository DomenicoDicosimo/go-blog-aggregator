-- name: GetAllPermissionsForUser :many
SELECT permissions.code
FROM permissions
INNER JOIN users_permissions ON users_permissions.permissions_id = permissions.id
INNER JOIN users ON users_permissions.user_id = users.id
WHERE users.id = $1;

-- name: GrantPermissionToUser :exec
INSERT INTO users_permissions (user_id, permissions_id)
SELECT @user_id::uuid, permissions.id 
FROM permissions 
WHERE permissions.code = ANY(@codes::text[]);