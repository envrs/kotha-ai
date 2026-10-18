-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tool_calls (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    message_id TEXT,
    tool_call_id TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    input TEXT,
    output TEXT,
    is_error INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,  -- Unix timestamp in milliseconds
    FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE CASCADE,
    FOREIGN KEY (message_id) REFERENCES messages (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tool_calls_session_id ON tool_calls (session_id);
CREATE INDEX IF NOT EXISTS idx_tool_calls_tool_call_id ON tool_calls (tool_call_id);
CREATE INDEX IF NOT EXISTS idx_tool_calls_message_id ON tool_calls (message_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tool_calls_message_id;
DROP INDEX IF EXISTS idx_tool_calls_tool_call_id;
DROP INDEX IF EXISTS idx_tool_calls_session_id;
DROP TABLE IF EXISTS tool_calls;
-- +goose StatementEnd