-- name: CreateVerificationToken :exec
INSERT INTO email_verification_tokens (token_hash, user_id, expires_at)
VALUES ($1, $2, $3);

-- name: ConsumeVerificationToken :one
UPDATE email_verification_tokens
SET used_at = NOW()
WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
RETURNING user_id;

-- name: VerificationTokenExpired :one
SELECT EXISTS (
    SELECT 1
    FROM email_verification_tokens
    WHERE token_hash = $1 AND expires_at <= NOW()
) AS expired;
