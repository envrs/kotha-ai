# Kotha

Kotha is a terminal-first AI assistant for software development. It provides an interactive chat-style workflow, project-aware context handling, LSP integration, and support for multiple LLM providers from a single local CLI.

This repository contains the Go implementation of Kotha, including the terminal UI, HTTP server, API client, configuration schema, and internal application logic.

## Features

- Terminal-based interactive AI assistant
- Non-interactive prompt execution for scripts and automation
- Local project context awareness
- LSP-backed coding support
- HTTP server mode for remote or background usage
- API subcommands for session management and requests
- Multi-provider model support
- JSON schema generation for configuration validation

## Project Structure

- `cmd/` — CLI entrypoints and subcommands
  - `cmd/root.go` — main CLI definition
  - `cmd/server/` — background server command
  - `cmd/api/` — HTTP client/API subcommands
  - `cmd/schema/` — configuration schema generation
- `internal/` — application internals, LLM integration, config, TUI, DB, sessions, and service logic
- `sdk/` — generated or shared SDK-related code
- `scripts/` — repo maintenance and generation helpers
- `main.go` — application bootstrap
- `openapi.yaml` — API specification
- `kotha-schema.json` — generated configuration schema
- `install` — helper script for installing the binary
- `Makefile` — build, test, lint, and release commands

## Requirements

- Go 1.24+
- A supported LLM provider configuration (for example OpenAI, Anthropic, Gemini, Groq, OpenRouter, Azure, Bedrock, or Vertex AI)
- Optional: `gopls` or another LSP server for IDE-like functionality

## Installation

### From source

```bash
git clone https://github.com/kothagpt/kotha-ai.git
cd kotha-ai
make build