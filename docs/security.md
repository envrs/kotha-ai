# Security Audit

This document summarizes the security posture of Kotha and the checks run on every change.

## Secrets handling

- API keys and provider credentials are stored in the config file and read from the environment. They are never written to logs, the database, or session exports.
- The HTTP server authenticates requests via the `X-API-Key` header (see `cmd/server/middleware.go`). Requests without a valid key are rejected with 401.
- Tool execution is gated by the `permission` service; destructive operations require explicit approval in interactive mode.

## Input validation

- All HTTP request bodies are decoded with `json.NewDecoder` and rejected on malformed JSON.
- Path parameters (`session_id`, `share_id`) are validated as non-empty before use.
- The agent loop passes user prompts through the LLM provider; untrusted file contents are wrapped in clearly delimited context blocks.

## Sandboxing

- File tools operate under the configured working directory; paths outside it are rejected.
- Shell execution is restricted to the allow-list defined in the tool registry.

## Data retention

- Sessions, messages, tool calls, and symbols are stored in SQLite under the configured data directory.
- Deleting a session cascades to messages, files, tool calls, and symbols.

## Audit checklist

- [x] API keys are never logged
- [x] Request bodies are validated before use
- [x] Path parameters are validated
- [x] Destructive tools require permission
- [x] File access is restricted to the working directory
- [x] Session deletion cascades correctly

## Running the audit

```bash
go vet ./...
golangci-lint run ./...
go test ./...
```