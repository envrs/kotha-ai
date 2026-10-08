# 📋 Comprehensive Build Plan

## Phase 1: Foundation (Weeks 1–2)

### Core App Initialization
- [x] Finalize `app.New()` to initialize all subsystems (DB, config, providers, LSP)
- [x] Implement graceful shutdown in `app.Shutdown()`
- [x] Error recovery for panics in agent goroutines

### Multi-Provider LLM Integration
- [x] Implement provider factory in `llm/provider/` (OpenAI, Anthropic, Gemini, GROQ, etc.)
- [x] Token counting & cost tracking per model in `llm/runtime/`
- [x] Model routing logic in `llm/route/` (fallback, cost optimization)
- [x] Streaming response handling for all providers

### Configuration System
- [x] Config file loading (YAML/JSON) with Viper
- [x] JSON schema generation (`cmd/schema/`) for IDE autocompletion
- [x] Environment variable overrides
- [x] Config validation

## Phase 2: LLM Agent & Tools (Weeks 3–4)

### Agentic Loop
- [x] `llm/agent/` — implement reasoning, tool calling, retry logic
- [x] Tool definitions in `llm/schema/` (structured tool specs)
- [x] Built-in tools: file read/write, code search, git commands
- [x] MCP tool integration: fetch and invoke external tools

### Project Context
- [x] Parse `.gitignore`, respect file exclusions
- [x] Scan repo for language-specific files (detect Go, Python, JS, etc.)
- [ ] Extract code symbols (classes, functions, imports) via LSP
- [x] Build context window efficiently (token budgeting)

### LSP Integration
- [x] `internal/lsp/` — spawn LSP servers per language
- [x] Hover info, go-to-definition, code completion
- [x] Diagnostics/errors extraction
- [ ] Symbol indexing for cross-file references

## Phase 3: Session & Persistence (Week 5)

### Database Schema
- [x] Sessions table: ID, name, created, updated, status
- [x] Messages table: session_id, role, content, timestamp
- [ ] Tools/calls table: track tool invocations and results

### Session Management
- [ ] Create new session, resume session, list sessions
- [ ] Save conversation history to SQLite
- [x] Export session as JSON/text
- [ ] Share links (local URL or cloud integration)

### History & Recall
- [ ] Search past sessions (semantic or keyword)
- [ ] Recall context from previous runs
- [ ] Diff view for changes across sessions

## Phase 4: Terminal UI (Week 6)

### Bubble Tea Components
- [ ] Main model/view/update architecture
- [ ] Input field (multiline, with history)
- [ ] Message list (scrollable, with syntax highlighting)
- [ ] Status bar (provider, model, tokens, session info)
- [ ] Help modal, keybindings display

### Interactivity
- [ ] Streaming output (real-time token rendering)
- [ ] Copy/paste support
- [ ] Interrupt (Ctrl+C) to cancel ongoing request
- [ ] Session switching without restart
- [ ] Scroll through history

### Theming
- [ ] Built-in themes (Catppuccin, Dracula, Flexoki, Gruvbox, etc.)
- [ ] Custom colors via config

## Phase 5: HTTP Server & API (Week 7)

### OpenAPI-Driven API
- [ ] Routes: `/sessions`, `/ask`, `/stop`, `/history`
- [x] Middleware: auth (GitHub OAuth or API key), logging, rate limiting
- [ ] WebSocket support for streaming responses (optional)

### Client CLI
- [ ] `kotha api ask` — send prompt to server
- [ ] `kotha api stop` — cancel ongoing agent
- [ ] `kotha api session` — list/resume/export sessions

### Server Daemon
- [ ] Background process mode (`--daemon`, `--pidfile`)
- [ ] Log file output
- [ ] Health check endpoint

## Phase 6: Desktop App (Week 8)

### Electron/Tauri Wrapper (or native app)
- [ ] Package Kotha backend as binary or service
- [ ] Build native UI (React/Vue)
- [ ] Communicate via HTTP to local Kotha server
- [ ] Tray icon, auto-start, updater

## Phase 7: IDE Extensions (Weeks 9–10)

### VS Code Extension
- [ ] WebView-based UI or sidebar
- [ ] Connect to local Kotha server
- [ ] Context menu: "Ask Kotha"
- [ ] Inline code suggestions (optional)

### JetBrains Plugin (optional)
- [ ] Similar architecture; IJ plugin SDK

## Phase 8: Deployment & Documentation (Week 11)

### Release Pipeline
- [ ] Goreleaser config for multi-platform binaries (Linux, macOS, Windows)
- [ ] Homebrew formula, AUR package
- [ ] Docker image

### Documentation
- [ ] README with quick start
- [ ] Configuration guide
- [ ] Architecture docs
- [ ] API docs (auto-generated from OpenAPI)
- [ ] Troubleshooting guide

### Testing & Hardening
- [ ] Integration tests (mock LLM, file system)
- [ ] Performance benchmarks
- [ ] Security audit (input validation, secrets handling)

## Phase 9: Advanced Features (Week 12+)

### Multi-Session Parallelization
- [ ] Run multiple agents on same project simultaneously
- [ ] Coordination/locking (avoid file conflicts)
- [ ] Resource pooling (shared LSP servers, LLM connections)

### Reasoning & Advanced Models
- [ ] Support OpenAI o1-preview, Claude 3 extended thinking
- [ ] Custom thinking budgets

### GitHub Integration
- [ ] OAuth login to use Copilot plan
- [ ] PR review assistant
- [ ] Issue resolution workflow

### Analytics & Observability
- [ ] Usage tracking (optional, privacy-respecting)
- [ ] OpenTelemetry integration for tracing
- [ ] Cost dashboard

---

## 🔑 Critical Considerations

| Aspect | Notes |
|---|---|
| **Error Recovery** | Supervise agent goroutines with exponential backoff; restart on panic without losing session state |
| **Token Efficiency** | Implement smart context windowing; drop old messages before truncating project files |
| **LSP Reliability** | Handle LSP crashes gracefully; fall back to simple regex parsing if LSP unavailable |
| **Security** | Never log API keys; validate all user input; sandbox tool execution |
| **Performance** | Lazy-load LSP servers only when needed; cache symbol index; stream responses to avoid memory bloat |
| **Privacy** | Option to run entirely offline (local LLM via Ollama); audit data retention policies |

---

## 🎯 MVP Checklist (for production readiness)

- [ ] CLI works in interactive and non-interactive modes
- [ ] Multi-provider LLM support (at least OpenAI, Anthropic)
- [ ] Session persistence and history
- [ ] Basic LSP integration (Go, Python, JavaScript)
- [ ] Terminal UI is responsive and user-friendly
- [ ] HTTP server mode works
- [ ] Configuration via YAML/JSON
- [ ] Comprehensive error messages
- [ ] Unit/integration test coverage >60%
- [ ] Documentation is complete
- [ ] Release binaries for macOS, Linux, Windows
