-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS symbols (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    "range" TEXT NOT NULL,
    container_name TEXT,
    language TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_symbols_session_id ON symbols (session_id);
CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols (name);
CREATE INDEX IF NOT EXISTS idx_symbols_file_path ON symbols (file_path);
CREATE INDEX IF NOT EXISTS idx_symbols_language ON symbols (language);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_symbols_language;
DROP INDEX IF EXISTS idx_symbols_file_path;
DROP INDEX IF EXISTS idx_symbols_name;
DROP INDEX IF EXISTS idx_symbols_session_id;
DROP TABLE IF EXISTS symbols;
-- +goose StatementEnd