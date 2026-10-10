-- name: AppendOutboxEvent :exec
INSERT INTO outbox_events (id, type, aggregate, payload, created_at)
VALUES ($1, $2, $3, $4, $5);
