# Architecture

Kotha is a terminal-first AI coding assistant written in Go. It runs as either an interactive TUI, a background HTTP server, or a CLI that talks to the server.

## High-level overview

```
┌─────────────┐     ┌──────────────┐
│  TUI / CLI  │────▶│  HTTP Server │────▶ LLM provider
└─────────────┘     └──────────────┘
        ▲                  │
        │           ┌──────┴───────┐
        │           │   Agent      │
        │           │   + Tools    │
        │           └──────┬───────┘
        │                  │
        ▼                  ▼
┌─────────────┐   ┌────────────────┐
│  Config      │   │  DB (SQLite)   │
│  + LSP       │   │  + Sessions    │
│  + MCP       │   │  + Messages    │
└─────────────┘   │  + Tool calls  │
                  │  + Symbols     │
                  └────────────────┘
```

## Packages

| Package | Responsibility |
|---|---|
| `cmd/` | CLI entrypoints (`cmd/root.go`, `cmd/server/`, `cmd/api/`, `cmd/schema/`) |
| `internal/app/` | Application bootstrap, LSP client lifecycle, symbol indexing |
| `internal/db/` | SQLite schema, generated queries (sqlc-style), migrations |
| `internal/session/` | Session lifecycle, persistence |
| `internal/message/` | Message domain and part marshalling |
| `internal/toolcall/` | Tool invocation recording and querying |
| `internal/symbol/` | Code symbol persistence and search |
| `internal/history/` | Recall, diff, keyword search across sessions |
| `internal/llm/` | Provider factory, agent loop, tool schemas, MCP integration |
| `internal/lsp/` | LSP client, protocol types, workspace watcher |
| `internal/tui/` | Bubble Tea UI components, themes, layout |
| `internal/config/` | Config loading (Viper), schema generation, validation |
| `internal/permission/` | Permission gating for tool execution |

## Data flow

1. User sends a prompt via TUI or HTTP `/v1/sessions/{id}/ask`.
2. The agent loop (`internal/llm/agent/`) calls the LLM provider with the project context.
3. The LLM returns tool calls; the agent executes them through the permission service.
4. Tool calls and results are recorded in `tool_calls` and surfaced via `GET /v1/sessions/{id}/tool_calls`.
5. Messages and parts are persisted to SQLite.

## LSP lifecycle

Each configured language spawns an LSP server on app start. The client opens files on demand, requests diagnostics and document symbols, and indexes symbols into the `symbols` table for cross-file reference lookups. LSP clients are restarted on crash via `restartLSPClient`.

## HTTP API

The server exposes a REST API under `/v1/`. See `openapi.yaml` for the SDK surface and `cmd/server/handlers.go` for request handlers. Streaming endpoints use Server-Sent Events (`/ask/stream`) or WebSocket (`/ask/ws`).