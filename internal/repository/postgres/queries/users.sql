-- name: CreateUser :exec
INSERT INTO users (id, email, email_verified, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetUserByEmail :one
SELECT id, email, email_verified, status, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, email_verified, status, created_at, updated_at
FROM users
WHERE id = $1;

-- name: UpdateUser :execrows
UPDATE users
SET email = $2, email_verified = $3, status = $4, updated_at = $5
WHERE id = $1;
