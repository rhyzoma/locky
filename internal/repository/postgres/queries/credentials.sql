-- name: CreateCredential :exec
INSERT INTO password_credentials (user_id, password_hash, updated_at)
VALUES ($1, $2, $3);

-- name: GetCredentialByUserID :one
SELECT user_id, password_hash, updated_at
FROM password_credentials
WHERE user_id = $1;

-- name: UpdateCredential :exec
UPDATE password_credentials
SET password_hash = $2, updated_at = $3
WHERE user_id = $1;
