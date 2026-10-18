# @ai/ Restructure — Refactoring Plan

## Executive Summary

The `@ai/` module (`github.com/kothagpt/kotha/ai`) is a standalone TUI AI code assistant
that currently lives in the same repository as the superkit web framework
(`github.com/khulnasoft/superkit`) but is a completely separate Go module with no
shared code. This plan restructures `@ai/` to be more maintainable, testable, and
aligned with Go best practices — borrowing proven patterns from superkit where
applicable.

**Scope:** `ai/` directory only. Root superkit framework is out of scope.

**Current state:** ~100 Go source files, 4 test files, ~2.5% test coverage.
High coupling via global singletons, a God-object `App`, and a 480-line TUI switch.

**Goal:** Modular architecture with dependency injection, clear layering, and ≥60%
unit-testable code.

---

## Current Architecture

```
ai/
├── cmd/                    # Cobra CLI entrypoint
│   ├── root.go            # Wire-up: loads config, DB, app, TUI
│   └── schema/            # JSON schema generator
├── internal/
│   ├── app/               # ⚠️ GOD OBJECT — wires all services
│   ├── config/            # ⚠️ GLOBAL SINGLETON — config.Get() everywhere
│   ├── db/                # sqlc-generated SQLite queries
│   ├── diff/              # Unified diff + patch processing
│   ├── fileutil/          # File discovery, rg/fzf wrappers
│   ├── format/            # Output formatters (text/json)
│   ├── history/           # File version tracking
│   ├── llm/               # ⚠️ LARGE SUBSYSTEM — providers, agent, tools, prompts
│   │   ├── models/        # Model catalog + metadata
│   │   ├── provider/      # Provider client implementations
│   │   ├── agent/         # Agent runtime + stream processing
│   │   ├── tools/         # 14 tool implementations
│   │   └── prompt/        # System prompt templates
│   ├── logging/           # ⚠️ GLOBAL SINGLETON — defaultLogData + package funcs
│   ├── lsp/               # LSP client + protocol types
│   ├── message/           # Message domain (CRUD + serialization)
│   ├── permission/        # Permission request/grant system
│   ├── pubsub/            # Generic in-process event broker
│   ├── session/           # Session domain (CRUD)
│   ├── tui/               # ⚠️ MONOLITHIC — 480-line Update + deep nesting
│   │   ├── theme/
│   │   ├── styles/
│   │   ├── layout/
│   │   ├── page/
│   │   └── components/
│   └── version/           # Build version string
├── main.go                # Entry: recover panic + cmd.Execute()
├── go.mod                 # Separate module: github.com/kothagpt/kotha/ai
└── kotha-schema.json      # Config JSON schema
```

### Dependency Graph (simplified)

```
cmd/root.go
  └── app.App (God object)
        ├── config.Get()  ─────────────────────┐
        ├── db.Connect()                        │
        ├── session.Service                     │
        ├── message.Service                     │
        ├── history.Service                     │
        ├── permission.Service                  │
        ├── llm/agent.Service                   │
        │     ├── config.Get() ─────────────────┤
        │     ├── message.Service                │
        │     ├── session.Service                │
        │     ├── permission.Service             │
        │     └── llm/tools ─────────────────────┤
        │           ├── fileutil                 │
        │           ├── lsp.Client               │
        │           ├── history.Service          │
        │           ├── permission.Service       │
        │           └── tui/components/dialog    │ ← ⚠️ downward dep
        └── lsp.Clients (map[string]*lsp.Client)
              └── config.Get() ─────────────────┘

tui/tui.go
  └── app.App + {config, session, message, permission, agent, logging, ...}
```

---

## Key Problems

### 1. Global Singleton — `config` (HIGH severity)
- `config.Get()` called from 30+ files across 10+ packages
- `Config` is a mutable global; tests cannot run in parallel
- `Load()` is 980 lines with viper setup, defaults, validation, and provider detection
- `setProviderDefaults` is a 130-line if/else chain
- **Impact:** Impossible to run tests in parallel, hidden dependencies, hard to reason about state

### 2. Global Singleton — `logging` (HIGH severity)
- `defaultLogData` is a global singleton with pubsub
- `MessageDir` and `sessionLogMutex` are package-level globals
- No `Logger` interface; every consumer calls `logging.Info(...)` directly
- **Impact:** Tests pollute each other's log state, cannot inject test logger

### 3. God Object — `app.App` (HIGH severity)
- Holds 7+ service references, LSP client map, and concurrency primitives
- Constructor launches background goroutines for LSP initialization
- Passed as a concrete type everywhere, especially to the TUI
- **Impact:** Hard to test, violates SRP, any change to a service ripples to all consumers

### 4. TUI Monolithic Update (MEDIUM severity)
- `appModel.Update()` is a 480-line switch on message types
- Deep coupling to `app.App` fields (`a.app.Sessions`, `a.app.Permissions`, etc.)
- ~20 boolean flags for dialog visibility — state machine as ad-hoc booleans
- **Impact:** Hard to add new features, impossible to unit test UI logic in isolation

### 5. Downward Dependency — `llm/tools` → `tui/components/dialog` (MEDIUM severity)
- Domain tools import UI dialog components for completion providers
- Violates layering: domain should not depend on presentation
- **Impact:** Cannot test tools without TUI, circular dependency risk

### 6. Context Misuse (MEDIUM severity)
- `context.Background()` used in `chat.sendMessage`, `permission.Request`, `agent.Summarize`
- Parent context cancellation and timeouts are lost
- **Impact:** Goroutine leaks, unresponsive UI, resource waste

### 7. No Interfaces for External Systems (MEDIUM severity)
- LSP client consumed as concrete `*lsp.Client`
- LLM providers created internally with no abstraction boundary
- File operations use direct `os.ReadFile`/`os.WriteFile`
- **Impact:** Cannot write unit tests without real LSP servers or filesystem

### 8. Poor Test Coverage (HIGH severity)
- Only 4 test files for ~100 source files (~2.5% coverage, threshold is 28%)
- Existing tests only cover: prompt templates, theme colors, ls tool, custom commands
- **Impact:** High regression risk, slow development velocity

---

## Superkit Patterns to Adopt

| Superkit Pattern | Source | Application in @ai/ |
|------------------|--------|---------------------|
| `Kit`-centered handler | `kit/kit.go` | `AppController` interface for TUI callbacks |
| Service + interface | `bootstrap/plugins/auth/` | Explicit service interfaces for all domain services |
| Plugin self-containment | `bootstrap/plugins/auth/` | Tool plugins as self-contained modules |
| Async event system | `event/event.go` | Replace raw pubsub with structured event bus |
| Middleware composition | `kit/middleware/` | Middleware for agent tool execution pipeline |
| Configuration builder | `bootstrap/app/conf/` | Split `config.Load` into composable stages |
| Generic containers | `kit/container/` | Thread-safe generic collections for test utilities |
| Retry/backoff | `kit/retry/` | Retry for LLM provider calls and LSP initialization |
| Context-key pattern | `kit/kit.go` AuthKey | Type-safe context keys for session/message IDs |

---

## Refactoring Phases

---

### Phase 0: Foundation (Week 1-2) — Eliminate Global State

**Goal:** Remove `config` and `logging` singletons; introduce injectable interfaces.

#### 0a. Introduce `config.Provider` interface
```go
// internal/config/provider.go
type Provider interface {
    Get() *Config
    WorkingDirectory() string
    ProviderFor(name string) (*Provider, error)
    UpdateAgentModel(name, modelID string) error
    Reload() error
}

// Default implementation wraps existing global
type defaultProvider struct { cfg *Config }

func (p *defaultProvider) Get() *Config { return p.cfg }
// ...
```

**Changes:**
- `internal/config/config.go` — Keep `Load()` as constructor for `defaultProvider`, but don't set global `cfg`
- `internal/config/init.go` — Package-level `Get()` becomes deprecated wrapper
- All 30+ callers of `config.Get()` → inject `config.Provider`
- `cmd/root.go` — Loads config, passes provider to `app.New(ctx, conn, provider)`

**Files modified:** `config/*.go`, `app/app.go`, `llm/agent/*.go`, `lsp/*.go`, `tui/*.go`, `permission/*.go`, `diff/*.go`, `logging/*.go`, `cmd/root.go`

**Risk:** High — touches every package. Do in small PRs per package.

#### 0b. Introduce `logging.Logger` interface
```go
// internal/logging/interface.go
type Logger interface {
    Info(msg string, args ...any)
    Debug(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    RecoverPanic(component string, onRecover func())
    AppendToSessionLogFile(sessionID, msg string) error
}

// Default implementation wraps existing package funcs
type defaultLogger struct{}
```

**Changes:**
- `internal/logging/*.go` — Add `NewDefaultLogger()` factory, keep package funcs as convenience
- All callers → inject `logging.Logger` (same sweep as config)
- `defaultLogData` remains for backward compat but is deprecated

**Files modified:** Same sweep as config + `logging/*.go`

**Risk:** Medium — follows same pattern as config change.

#### 0c. Introduce `event.EventBus` (optional, can defer)
```go
// internal/event/bus.go  (new)
type Bus interface {
    Publish(topic string, event any) error
    Subscribe(topic string, handler HandlerFunc) Subscription
}

type HandlerFunc func(ctx context.Context, event any) error
```

**Changes:**
- Replace raw `pubsub.Broker[T]` embedding with `event.Bus` subscriptions
- Keeps pubsub as implementation detail

**Defer to Phase 3** if Phase 0a+0b take too long.

---

### Phase 1: Domain Layer Cleanup (Week 2-3)

**Goal:** Separate repository from notifier; add interfaces for testability.

#### 1a. Split `session.Service` into `SessionRepository` + `SessionNotifier`
```go
// internal/session/repository.go
type Repository interface {
    Create(ctx context.Context, s *Session) error
    Get(ctx context.Context, id string) (*Session, error)
    List(ctx context.Context) ([]Session, error)
    Save(ctx context.Context, s *Session) error
    Delete(ctx context.Context, id string) error
}

// internal/session/notifier.go
type Notifier interface {
    Subscribe(ctx context.Context) <-chan pubsub.Event[Session]
}

// internal/session/service.go — composes both
type Service interface {
    Repository
    Notifier
}
```

**Same pattern for:** `message`, `history`, `permission`

#### 1b. Add `FileStore` interface for file operations
```go
// internal/fileutil/store.go
type FileStore interface {
    Read(path string) ([]byte, error)
    Write(path string, data []byte) error
    Glob(pattern string) ([]string, error)
    Stat(path string) (os.FileInfo, error)
}

type defaultStore struct{}
```

**Changes:**
- `diff/patch.go` — inject `FileStore` instead of `os.ReadFile`/`os.WriteFile`
- `llm/tools/*.go` — inject `FileStore` for file operations
- `fileutil/fileutil.go` — implement `defaultStore`

#### 1c. Add `LSPClient` interface
```go
// internal/lsp/interface.go
type Client interface {
    Initialize(ctx context.Context, rootURI string) error
    OpenFile(ctx context.Context, path string, content []byte) error
    NotifyChange(ctx context.Context, path string, content []byte) error
    CloseFile(ctx context.Context, path string) error
    Diagnostics(ctx context.Context, path string) ([]Diagnostic, error)
    Shutdown(ctx context.Context) error
    IsReady() bool
}
```

**Changes:**
- `internal/lsp/client.go` — `Client` struct implements `Client` interface
- `internal/app/lsp.go` — inject interface, add `NewLSPManager()`
- Tests can use `FakeLSPClient`

---

### Phase 2: Application Layer Decomposition (Week 3-4)

**Goal:** Replace `App` God object with focused services.

#### 2a. Define `Application` interface
```go
// internal/app/interface.go
type Application interface {
    Sessions() session.Service
    Messages() message.Service
    History() history.Service
    Permissions() permission.Service
    Agent() agent.Service
    LSP() LSPManager
    Shutdown(ctx context.Context) error
}
```

#### 2b. Extract `LSPManager` service
```go
// internal/app/lsp.go
type LSPManager interface {
    GetClient(language string) (lsp.Client, error)
    InitializeAll(ctx context.Context) error
    Shutdown(ctx context.Context) error
}
```

#### 2c. Keep `App` struct but make it implement `Application`
```go
// internal/app/app.go
type App struct {
    sessions    session.Service
    messages    message.Service
    // ... private fields, no public fields
}

func (a *App) Sessions() session.Service    { return a.sessions }
func (a *App) Messages() message.Service    { return a.messages }
// ...
```

**Result:** TUI depends on `Application` interface, not concrete `*App`.

---

### Phase 3: LLM Layer Improvements (Week 4-5)

**Goal:** Decouple tools from UI, improve provider abstraction.

#### 3a. Remove `llm/tools` → `tui/components/dialog` dependency
- The `sourcegraph` tool imports dialog types for completion UI
- Extract completion into a callback:
```go
// internal/llm/tools/completion.go
type CompletionProvider interface {
    Complete(ctx context.Context, query string) ([]string, error)
}

// sourcegraph.go no longer imports tui
type SourcegraphTool struct {
    completion CompletionProvider  // injected, nil in non-interactive mode
}
```
- `tui` registers itself as the completion provider at app startup

#### 3b. Add `LLMProvider` interface boundary
```go
// internal/llm/provider/interface.go
type Provider interface {
    SendMessages(ctx context.Context, msgs []Message) (*Response, error)
    StreamResponse(ctx context.Context, msgs []Message) (<-chan StreamChunk, error)
    Models() []string
    Name() string
}
```
- Currently `provider.Provider` exists but is generic; standardize it
- Create `FakeProvider` for tests

#### 3c. Extract tool execution pipeline
```go
// internal/llm/tools/pipeline.go
type Pipeline struct {
    store       FileStore
    permission  permission.Service
    lsp         lsp.Client
    completion  CompletionProvider  // nil = non-interactive
    maxRetries  int
}

func (p *Pipeline) Execute(ctx context.Context, tool BaseTool, call ToolCall) (ToolResponse, error)
```
- Centralize retry, permission checks, and error handling

---

### Phase 4: TUI Refactoring (Week 5-6)

**Goal:** Decouple TUI from domain, reduce Update complexity.

#### 4a. Introduce `AppController` interface
```go
// internal/tui/controller.go
type AppController interface {
    CreateSession(ctx context.Context, title string) (session.Session, error)
    SendMessage(ctx context.Context, sessionID, content string) (message.Message, error)
    ListSessions(ctx context.Context) ([]session.Session, error)
    SwitchSession(id string) error
    GrantPermission(ctx context.Context, req permission.PermissionRequest) error
    DenyPermission(ctx context.Context, req permission.PermissionRequest) error
    // ... only methods the TUI actually calls
}
```

**Changes:**
- `tui/tui.go` — depends on `AppController`, not `app.App`
- `app/app.go` — implements `AppController`
- `cmd/root.go` — creates controller, passes to `tui.New(controller)`

#### 4b. Replace boolean flags with state machine
```go
// internal/tui/state.go
type AppState int

const (
    StateNormal AppState = iota
    StatePermissionDialog
    StateHelpDialog
    StateQuitDialog
    StateModelDialog
    // ...
)

type appState struct {
    current    AppState
    previous   AppState
    dialogData any  // type-safe dialog payload
}
```

#### 4c. Extract Update handlers into methods
```go
// internal/tui/handlers.go
type UpdateHandler struct {
    controller AppController
    model      *appModel
}

func (h *UpdateHandler) HandleChatMsg(msg tea.Msg) tea.Cmd { ... }
func (h *UpdateHandler) HandlePermissionMsg(msg tea.Msg) tea.Cmd { ... }
func (h *UpdateHandler) HandleAgentEvent(msg tea.Msg) tea.Cmd { ... }
```
- `appModel.Update()` becomes a dispatcher:
```go
func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch m.state.current {
    case StateNormal:
        return h.HandleChatMsg(msg)
    case StatePermissionDialog:
        return h.HandlePermissionMsg(msg)
    // ...
    }
}
```

---

### Phase 5: Infrastructure Improvements (Week 6-7)

#### 5a. Improve LSP client testability
- Create `FakeLSPClient` implementing `lsp.Client` interface
- Add `LSPRecorder` for integration tests that records all calls

#### 5b. Add `Retry` middleware for agent tools
- Borrow `kit/retry/` pattern:
```go
// internal/llm/tools/retry.go
func WithRetry(maxAttempts int, baseDelay time.Duration) ToolMiddleware {
    return func(next BaseTool) BaseTool {
        return &retryTool{next, maxAttempts, baseDelay}
    }
}
```

#### 5c. Add configuration builder pattern
```go
// internal/config/builder.go
type Builder struct {
    workingDir string
    debug      bool
    overrides  map[string]any
}

func (b *Builder) Load() (*Config, error) {
    cfg := &Config{}
    b.applyDefaults(cfg)
    b.mergeFile(cfg)
    b.applyOverrides(cfg)
    b.validate(cfg)
    return cfg, nil
}
```
- Replaces 130-line `setProviderDefaults` chain with strategy map:
```go
var providerDefaults = map[string]func(cfg *Config){
    "OPENAI_API_KEY":    setOpenAIDefaults,
    "ANTHROPIC_API_KEY": setAnthropicDefaults,
    // ...
}
```

---

### Phase 6: Testing Infrastructure (Week 7-8)

**Goal:** Increase test coverage from 2.5% to ≥60%.

#### 6a. Add test utilities
```go
// internal/testing/application.go
type FakeApplication struct {
    Sessions    *FakeSessionService
    Messages    *FakeMessageService
    History     *FakeHistoryService
    Permissions *FakePermissionService
    Agent       *FakeAgentService
}

func NewFakeApplication() *FakeApplication { ... }

// internal/testing/logger.go
type CaptureLogger struct {
    Messages []string
}

func (l *CaptureLogger) Info(msg string, args ...any) { l.Messages = append(...) }
// ...
```

#### 6b. Add test helpers for pubsub
```go
// internal/pubsub/testing.go
type TestBroker[T any] struct {
    mu    sync.Mutex
    items []Event[T]
}

func (t *TestBroker[T]) Drain() []Event[T] { ... }
func (t *TestBroker[T]) Clear()           { ... }
```

#### 6c. Priority test additions
| Component | Current Tests | Priority | Approach |
|-----------|--------------|----------|----------|
| `config.Load` | 0 | HIGH | Table-driven, mock viper |
| `diff.ParseUnifiedDiff` | 0 | HIGH | Golden file tests |
| `llm/tools/edit` | 0 | HIGH | In-memory filesystem |
| `session.Service` | 0 | HIGH | In-memory SQLite + TestBroker |
| `message.Service` | 0 | HIGH | In-memory SQLite + TestBroker |
| `agent.Service` | 0 | MEDIUM | FakeProvider + FakePermission |
| `tui.Update` | 0 | MEDIUM | Bubble Tea test harness + FakeController |
| `lsp.Client` | 0 | MEDIUM | FakeLSPClient |

---

### Phase 7: Optional — Superkit Convergence (Week 8+)

**Goal:** Evaluate whether `@ai/` should consume superkit patterns or stay independent.

#### Option A: Stay Independent (Recommended)
- `@ai/` is a CLI product; superkit is a web framework
- Borrow patterns (interfaces, middleware, event bus) but don't couple modules
- Maintain separate `go.mod` for independent release cycles

#### Option B: Extract Shared Kit
- Create `internal/kit/` within `@ai/` for shared utilities
- Potential candidates: `event.Bus`, `retry`, `config.Provider`, `logging.Logger`
- Similar to how `superkit/kit/` serves the web framework

#### Option C: Full Convergence
- Unlikely to be beneficial — TUI and web paradigms are fundamentally different
- Only consider if team wants a unified "Kotha Platform"

**Recommendation:** Option A — stay independent, adopt patterns.

---

## Migration Checklist

### Per-Package Migration Order
1. `config/` — Introduce `Provider` interface, eliminate global `cfg`
2. `logging/` — Introduce `Logger` interface, eliminate `defaultLogData`
3. `db/` — Already has `Querier` interface (sqlc-generated), add `TestQuerier`
4. `session/` — Split repository/notifier, add `FakeSessionService`
5. `message/` — Split repository/notifier, add `FakeMessageService`
6. `history/` — Split repository/notifier, add `FakeHistoryService`
7. `permission/` — Add timeout to `Request()`, add `FakePermissionService`
8. `llm/provider/` — Standardize `Provider` interface, add `FakeProvider`
9. `llm/agent/` — Inject `config.Provider`, `Logger`, remove `context.Background()`
10. `llm/tools/` — Inject `FileStore`, `CompletionProvider`, remove tui dependency
11. `lsp/` — Add `Client` interface, `FakeLSPClient`, inject config
12. `app/` — Implement `Application` interface, decompose responsibilities
13. `tui/` — Depend on `AppController` interface, refactor Update
14. `cmd/` — Wire everything with dependency injection

### Backward Compatibility Strategy
- Keep package-level convenience functions (e.g., `config.Get()`, `logging.Info()`) during transition
- Add `// Deprecated:` comments
- Remove in next major version after all callers migrated

### CI/CD Changes
- Add `go vet`, `staticcheck` to CI
- Enforce minimum coverage threshold (start at 30%, ramp to 60%)
- Add `golangci-lint` for comprehensive linting
- Parallelize test execution

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Breaking config API | HIGH | HIGH | Deprecation period, feature flags |
| TUI regressions | MEDIUM | HIGH | Manual testing, screenshot diffing |
| LSP initialization race | MEDIUM | HIGH | Integration tests with real LSP servers |
| LLM provider API changes | LOW | MEDIUM | Version-pin provider SDKs, contract tests |
| Migration too large | HIGH | MEDIUM | Phased approach, one package at a time |

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Unit test coverage | ~2.5% | ≥60% |
| Files importing `config.Get()` | ~30 | 0 (use interface) |
| `context.Background()` calls in domain | ~8 | 0 |
| TUI Update method length | 480 lines | ≤80 lines per handler |
| `app.App` public fields | 7+ services | 0 (interface-only) |
| Test files | 4 | ≥20 |
| Cyclomatic complexity (avg per func) | ~15 | ≤8 |
| Build time | ~8s | ≤5s (via module splitting) |

---

## Appendix: Superkit Patterns Reference

### Kit Pattern (`kit/kit.go`)
```go
type Kit struct {
    Response http.ResponseWriter
    Request  *http.Request
}
type HandlerFunc func(kit *Kit) error
```
**Adaptation for @ai/:** `AppController` as the central abstraction for TUI callbacks.

### Plugin Pattern (`bootstrap/plugins/auth/`)
```go
func InitializeRoutes(router chi.Router, deps Dependencies) {
    router.Route("/auth", func(r chi.Router) {
        r.Get("/login", HandleLogin(deps))
        // ...
    })
}
```
**Adaptation for @ai/:** Tool plugins register themselves with `ToolRegistry` at init time.

### Event Pattern (`event/event.go`)
```go
func Emit(topic string, event any) {
    select {
    case eventStream.eventChannel <- event:
    default: // drop if full
    }
}
```
**Adaptation for @ai/:** Replace raw pubsub with typed event bus; non-blocking publish with metrics.

### Middleware Pattern (`kit/middleware/`)
```go
func WithRequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := generateID()
        ctx := context.WithValue(r.Context(), requestIDKey{}, id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```
**Adaptation for @ai/:** Middleware for agent tool execution (auth, logging, retry).

---

## Progress Update — Phase 0 (In Progress)

### Completed: 0a. `config.Provider` Interface

**Files created:**
- `internal/config/provider.go` — `ConfigProvider` interface + `defaultProvider` implementation + `SetDefaultProvider()`/`GetDefaultProvider()`

**Files modified:**
- `internal/config/config.go` — `Load()` now returns `ConfigProvider`; backward-compatible `Get()`, `WorkingDirectory()`, `UpdateAgentModel()`, `UpdateTheme()` delegate to `GetDefaultProvider()`
- `internal/config/init.go` — `ShouldShowInitDialog()` and `MarkProjectInitialized()` delegate to `GetDefaultProvider().Get()`
- `internal/app/app.go` — `New()` accepts `config.ConfigProvider`; `initTheme()` takes provider parameter
- `internal/app/lsp.go` — `initLSPClients()`, `createAndStartLSPClient()`, `runWorkspaceWatcher()`, `restartLSPClient()` all accept provider parameter
- `cmd/root.go` — Passes `provider` from `config.Load()` to `app.New()`
- `internal/llm/prompt/prompt_test.go` — Updated to use returned provider
- `internal/llm/tools/ls_test.go` — Sets up default provider for tests
- `internal/tui/theme/theme_test.go` — Sets up default provider for tests

**Verification:**
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./...` — PASS (all 4 test packages)

**Key design decisions:**
- Kept global `cfg` variable for backward compatibility during migration
- Package-level functions (`config.Get()`, etc.) delegate to `GetDefaultProvider()`
- New code injects `ConfigProvider` interface; old code continues to work
- Tests can override the default provider via `SetDefaultProvider()`

### Completed: 0b. `logging.Logger` Interface

**Files created:**
- `internal/logging/interface.go` — `Logger` interface + `defaultLogger` implementation + `SetDefaultLogger()`/`GetDefaultLogger()`

**Files modified:**
- `internal/llm/agent/*.go` — inject `Logger` into agents, tools, and MCP tool construction
- `internal/lsp/client.go`, `internal/lsp/transport.go`, `internal/lsp/handlers.go` — inject `Logger` into LSP clients and transport
- `internal/lsp/watcher/watcher.go` — inject `Logger` into workspace watcher
- `internal/app/app.go`, `internal/app/lsp.go` — pass the default logger at the composition root

**Verification:**
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./...` — PASS

### Completed: 0c. LSP Dependency Injection

**Files modified:**
- `internal/lsp/client.go` — `Client` stores `ConfigProvider` and `Logger`; `NewClientWithConfig()` accepts both, with backward-compatible fallbacks
- `internal/lsp/transport.go` — standalone message functions accept provider/logger; client methods use their injected dependencies
- `internal/lsp/watcher/watcher.go` — watcher stores provider/logger and uses them for configuration and diagnostics
- `internal/lsp/handlers.go` — request/notification handlers receive the client dependencies they need
- `internal/app/lsp.go` — passes provider/logger into client and watcher construction

**Verification:**
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./...` — PASS

### Next Steps
- Migrate `tui` package to inject `ConfigProvider` and `Logger`
- Migrate `llm/tools` to inject `ConfigProvider` and `Logger`
- Migrate `permission`, `diff`, and remaining provider call sites
- Continue with Phase 1: Split domain services into Repository + Notifier
