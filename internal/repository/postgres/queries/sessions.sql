-- name: CreateSession :exec
INSERT INTO sessions (token_hash, id, user_id, expires_at, revoked)
VALUES ($1, $2, $3, $4, $5);

-- name: GetSessionByTokenHash :one
SELECT id, user_id, token_hash, expires_at, revoked
FROM sessions
WHERE token_hash = $1;

-- name: RevokeSessionForRotation :execrows
UPDATE sessions
SET revoked = TRUE
WHERE token_hash = $1 AND revoked = FALSE AND expires_at > NOW();

-- name: RevokeSession :execrows
UPDATE sessions
SET revoked = TRUE
WHERE token_hash = $1;

-- name: RevokeAllSessions :exec
UPDATE sessions
SET revoked = TRUE
WHERE user_id = $1;
