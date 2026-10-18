-- name: SearchSessions :many
SELECT DISTINCT s.*
FROM sessions s
LEFT JOIN messages m ON m.session_id = s.id
WHERE s.parent_session_id IS NULL
  AND (
    s.title LIKE ? COLLATE NOCASE
    OR m.parts LIKE ? COLLATE NOCASE
  )
ORDER BY s.created_at DESC;