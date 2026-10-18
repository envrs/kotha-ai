-- name: CreateToolCall :one
INSERT INTO tool_calls (
    id,
    session_id,
    message_id,
    tool_call_id,
    tool_name,
    input,
    output,
    is_error,
    duration_ms,
    created_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%s', 'now')
)
RETURNING *;

-- name: GetToolCall :one
SELECT *
FROM tool_calls
WHERE id = ? LIMIT 1;

-- name: GetToolCallByToolCallID :one
SELECT *
FROM tool_calls
WHERE tool_call_id = ? LIMIT 1;

-- name: ListToolCallsBySession :many
SELECT *
FROM tool_calls
WHERE session_id = ?
ORDER BY created_at ASC;

-- name: ListToolCallsByMessage :many
SELECT *
FROM tool_calls
WHERE message_id = ?
ORDER BY created_at ASC;

-- name: DeleteToolCall :exec
DELETE FROM tool_calls
WHERE id = ?;

-- name: DeleteSessionToolCalls :exec
DELETE FROM tool_calls
WHERE session_id = ?;