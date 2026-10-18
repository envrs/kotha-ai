package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kothagpt/kotha/internal/message"
)

// DeliveryMode controls how the prompt is delivered to the agent.
type DeliveryMode string

const (
	DeliveryDirect  DeliveryMode = "direct"  // Normal prompt delivery
	DeliveryStream  DeliveryMode = "stream"  // Stream response
	DeliveryWebhook DeliveryMode = "webhook" // Via webhook
)

// FileDraft represents a file prepared for admission (staged, not yet attached).
type FileDraft struct {
	ID       string    // Draft ID for idempotency
	Path     string    // Source path
	Name     string    // Original name
	Size     int64     // File size
	MIMEType string    // Detected MIME type
	Content  []byte    // File content (cached after preparation)
	Prepared bool      // Whether content has been read
	StagedAt time.Time // When the draft was staged
}

// AgentMention represents an @agent mention in the prompt.
type AgentMention struct {
	Name      string                 // Agent name (e.g., "coder", "reviewer")
	Metadata  map[string]any         // Additional mention metadata
	Raw       string                 // Raw mention text (e.g., "@coder")
	StartPos  int                    // Start position in text
	EndPos    int                    // End position in text
}

// SkillRef represents a skill reference in the prompt.
type SkillRef struct {
	Name     string                 // Skill name
	Args     map[string]any         // Skill arguments
	Raw      string                 // Raw skill reference (e.g., "!skill args")
	StartPos int                    // Start position in text
	EndPos   int                    // End position in text
}

// PromptRequest is the admission-time view of a user prompt, handed to
// interceptors before the prompt is persisted or sent to the LLM.
type PromptRequest struct {
	SessionID       string
	Text            string                 // Raw text (may contain mentions/skills)
	Files           []FileDraft            // Staged file drafts
	AgentMentions   []AgentMention         // Parsed @agent mentions
	Skills          []SkillRef             // Parsed skill references
	Metadata        map[string]any         // Arbitrary metadata
	Delivery        DeliveryMode           // Delivery mode
	AdmissionID     string                 // Unique ID for this admission attempt
	AlreadyAdmitted bool                   // Whether this ID was already admitted
	Prepared        bool                   // Whether preparation completed

	// Legacy fields for backward compatibility with Chain interceptors
	Content     string
	Attachments []message.Attachment
}

// PromptHook is a function that can transform a PromptRequest before admission.
// Hooks are called in registration order; each can mutate the request.
// Return an error to reject (will be wrapped in BlockedError).
type PromptHook func(ctx context.Context, req *PromptRequest) error

// HookRegistry manages prompt hooks with per-session scoping.
// Hooks registered globally run for all sessions; session-scoped hooks
// run only for their session and take precedence.
type HookRegistry struct {
	mu           sync.RWMutex
	global       []PromptHook
	sessionHooks map[string][]PromptHook
}

func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		sessionHooks: make(map[string][]PromptHook),
	}
}

// RegisterGlobal adds a hook that runs for all sessions.
func (r *HookRegistry) RegisterGlobal(hook PromptHook) {
	if hook == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.global = append(r.global, hook)
}

// RegisterSession adds a hook for a specific session.
// This implements the ctx.session.hook("prompt", ...) pattern.
func (r *HookRegistry) RegisterSession(sessionID string, hook PromptHook) {
	if hook == nil || sessionID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionHooks[sessionID] = append(r.sessionHooks[sessionID], hook)
}

// GetHooks returns all hooks for a session (global + session-scoped).
func (r *HookRegistry) GetHooks(sessionID string) []PromptHook {
	r.mu.RLock()
	defer r.mu.RUnlock()
	hooks := make([]PromptHook, 0, len(r.global)+len(r.sessionHooks[sessionID]))
	hooks = append(hooks, r.global...)
	if sessionHooks, ok := r.sessionHooks[sessionID]; ok {
		hooks = append(hooks, sessionHooks...)
	}
	return hooks
}

// ClearSession removes all session-scoped hooks for a session.
func (r *HookRegistry) ClearSession(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessionHooks, sessionID)
}

// PromptInterceptor is the legacy interface for backward compatibility.
type PromptInterceptor interface {
	Name() string
	InterceptPrompt(ctx context.Context, req *PromptRequest) error
}

// LegacyFunc creates a PromptInterceptor from a function for backward
// compatibility with the Chain API.
func LegacyFunc(name string, fn func(ctx context.Context, req *PromptRequest) error) PromptInterceptor {
	return &legacyFuncInterceptor{name: name, fn: fn}
}

type legacyFuncInterceptor struct {
	name string
	fn   func(ctx context.Context, req *PromptRequest) error
}

func (l *legacyFuncInterceptor) Name() string { return l.name }

func (l *legacyFuncInterceptor) InterceptPrompt(ctx context.Context, req *PromptRequest) error {
	return l.fn(ctx, req)
}

// Func creates a PromptHook from a function (legacy-compatible name).
func Func(name string, fn func(ctx context.Context, req *PromptRequest) error) PromptHook {
	return fn
}

// BlockedError rejects a prompt with a human-readable reason.
type BlockedError struct {
	Interceptor string
	Reason      string
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf("prompt blocked by %q: %s", e.Interceptor, e.Reason)
}

func Block(interceptor, reason string) *BlockedError {
	return &BlockedError{Interceptor: interceptor, Reason: reason}
}

// Chain runs registered interceptors sequentially (legacy API).
type Chain struct {
	mu           sync.RWMutex
	interceptors []PromptInterceptor
}

func NewChain() *Chain { return &Chain{} }

func (c *Chain) Use(i PromptInterceptor) {
	if i == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interceptors = append(c.interceptors, i)
}

func (c *Chain) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.interceptors)
}

func (c *Chain) Names() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	names := make([]string, 0, len(c.interceptors))
	for _, i := range c.interceptors {
		names = append(names, i.Name())
	}
	return names
}

func (c *Chain) Intercept(ctx context.Context, req *PromptRequest) error {
	c.mu.RLock()
	interceptors := make([]PromptInterceptor, len(c.interceptors))
	copy(interceptors, c.interceptors)
	c.mu.RUnlock()

	for _, i := range interceptors {
		if err := i.InterceptPrompt(ctx, req); err != nil {
			var blocked *BlockedError
			if errors.As(err, &blocked) {
				return err
			}
			return &BlockedError{Interceptor: i.Name(), Reason: err.Error()}
		}
	}
	return nil
}

// AdmissionStore tracks admitted prompt IDs for idempotency.
// Uses a simple in-memory store with TTL cleanup.
type AdmissionStore struct {
	mu      sync.RWMutex
	admitted map[string]time.Time
	ttl      time.Duration
}

func NewAdmissionStore(ttl time.Duration) *AdmissionStore {
	s := &AdmissionStore{
		admitted: make(map[string]time.Time),
		ttl:      ttl,
	}
	go s.cleanup()
	return s
}

func (s *AdmissionStore) cleanup() {
	ticker := time.NewTicker(s.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-s.ttl)
		for id, t := range s.admitted {
			if t.Before(cutoff) {
				delete(s.admitted, id)
			}
		}
		s.mu.Unlock()
	}
}

// MarkAdmitted marks an admission ID as admitted.
func (s *AdmissionStore) MarkAdmitted(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.admitted[id] = time.Now()
}

// IsAdmitted checks if an admission ID has already been admitted.
func (s *AdmissionStore) IsAdmitted(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.admitted[id]
	return ok
}

// PrepareContext holds state during prompt preparation.
type PrepareContext struct {
	Ctx         context.Context
	Req         *PromptRequest
	StagedFiles []*FileDraft
	Err         error
}

// PrepareStage represents a preparation stage.
type PrepareStage int

const (
	StageText       PrepareStage = iota // Text processing (mentions, skills)
	StageFiles                          // File reading/staging
	StageResolution                     // Attachment/skill resolution
	StageDelivery                       // Delivery mode setup
)

// PrepareFunc is a preparation step that can be interrupted and reverted.
type PrepareFunc func(ctx *PrepareContext, stage PrepareStage) error

// RevertFunc is called to undo a preparation step on failure/retry.
type RevertFunc func(ctx *PrepareContext, stage PrepareStage) error

// PreparationPipeline orchestrates staged prompt preparation with
// interruptibility and revert-on-failure semantics.
type PreparationPipeline struct {
	stages    []struct {
		name     string
		prepare  PrepareFunc
		revert   RevertFunc
		stage    PrepareStage
	}
	hooks       *HookRegistry
	admissions  *AdmissionStore
	mu          sync.Mutex
	inProgress  map[string]*PrepareContext // admissionID -> context
}

func NewPreparationPipeline(hooks *HookRegistry, admissions *AdmissionStore) *PreparationPipeline {
	return &PreparationPipeline{
		hooks:      hooks,
		admissions: admissions,
		inProgress: make(map[string]*PrepareContext),
	}
}

// AddStage adds a preparation stage.
func (p *PreparationPipeline) AddStage(name string, stage PrepareStage, prepare PrepareFunc, revert RevertFunc) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stages = append(p.stages, struct {
		name     string
		prepare  PrepareFunc
		revert   RevertFunc
		stage    PrepareStage
	}{name, prepare, revert, stage})
}

// Prepare runs the full preparation pipeline for a request.
// It is interruptible via ctx; on failure it reverts completed stages.
func (p *PreparationPipeline) Prepare(ctx context.Context, req *PromptRequest) error {
	// Check idempotency
	if req.AdmissionID != "" && p.admissions.IsAdmitted(req.AdmissionID) {
		req.AlreadyAdmitted = true
		return nil // Skip preparation for already-admitted
	}

	// Check if preparation already in progress for this ID
	p.mu.Lock()
	if _, ok := p.inProgress[req.AdmissionID]; ok {
		p.mu.Unlock()
		// Wait for existing preparation to complete
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(30 * time.Second):
			return fmt.Errorf("preparation timeout waiting for existing")
		}
	}
	pCtx := &PrepareContext{
		Ctx: ctx,
		Req: req,
	}
	p.inProgress[req.AdmissionID] = pCtx
	p.mu.Unlock()

	// Ensure cleanup
	defer func() {
		p.mu.Lock()
		delete(p.inProgress, req.AdmissionID)
		p.mu.Unlock()
	}()

	completedStages := []int{}
	for i, st := range p.stages {
		select {
		case <-ctx.Done():
			p.revertStages(pCtx, completedStages)
			return ctx.Err()
		default:
		}

		if err := st.prepare(pCtx, st.stage); err != nil {
			p.revertStages(pCtx, completedStages)
			return err
		}
		completedStages = append(completedStages, i)
	}

	// Mark as admitted
	if req.AdmissionID != "" {
		p.admissions.MarkAdmitted(req.AdmissionID)
	}
	req.Prepared = true
	return nil
}

func (p *PreparationPipeline) revertStages(pCtx *PrepareContext, completed []int) {
	for i := len(completed) - 1; i >= 0; i-- {
		st := p.stages[completed[i]]
		if st.revert != nil {
			_ = st.revert(pCtx, st.stage) // Best effort revert
		}
	}
}

// SessionHookContext is the context key for session-scoped hooks.
type sessionHookContextKey struct{}

// WithSessionHooks attaches a HookRegistry to the context.
func WithSessionHooks(ctx context.Context, registry *HookRegistry) context.Context {
	return context.WithValue(ctx, sessionHookContextKey{}, registry)
}

// GetHookRegistry retrieves the HookRegistry from context.
func GetHookRegistry(ctx context.Context) *HookRegistry {
	if r, ok := ctx.Value(sessionHookContextKey{}).(*HookRegistry); ok {
		return r
	}
	return nil
}

// SessionHookKey is the key used for session.hook("prompt", ...)
const SessionHookKey = "prompt"

// SessionHook allows registering hooks on a session via context.
// Usage: ctx.session.hook("prompt", hookFn) — attaches to session scope.
type SessionHook struct {
	sessionID string
	registry  *HookRegistry
}

func (h *SessionHook) Hook(key string, hook PromptHook) {
	if h.registry != nil && key == SessionHookKey {
		h.registry.RegisterSession(h.sessionID, hook)
	}
}

// NewSessionHook creates a SessionHook for a session.
func NewSessionHook(sessionID string, registry *HookRegistry) *SessionHook {
	return &SessionHook{sessionID: sessionID, registry: registry}
}

// RetryDecision represents the decision of a retry hook.
type RetryDecision int

const (
	// RetryDecisionRetry indicates the operation should be retried.
	RetryDecisionRetry RetryDecision = iota
	// RetryDecisionFail indicates the operation should fail without retry.
	RetryDecisionFail
	// RetryDecisionUseFallback indicates the operation should use a fallback (if available).
	RetryDecisionUseFallback
)

// RetryContext provides context to a retry hook about the failed operation.
type RetryContext struct {
	// SessionID is the ID of the session where the failure occurred.
	SessionID string
	// Attempt is the current attempt number (1-indexed).
	Attempt int
	// MaxAttempts is the maximum number of attempts allowed by the provider.
	MaxAttempts int
	// Error is the error that caused the failure.
	Error error
	// ProviderName is the name of the provider that failed.
	ProviderName string
	// ModelID is the model ID that was being used.
	ModelID string
	// IsStream indicates whether the failure occurred during streaming.
	IsStream bool
	// Metadata holds arbitrary metadata about the request.
	Metadata map[string]any
}

// RetryHook is called when a provider operation fails, allowing plugins to
// decide whether to retry, fail, or use a fallback.
// The hook should return a RetryDecision and optionally a modified error.
type RetryHook func(ctx context.Context, retryCtx *RetryContext) (RetryDecision, error)

// RetryHookRegistry manages retry hooks with per-session scoping.
type RetryHookRegistry struct {
	mu           sync.RWMutex
	global       []RetryHook
	sessionHooks map[string][]RetryHook
}

// NewRetryHookRegistry creates a new retry hook registry.
func NewRetryHookRegistry() *RetryHookRegistry {
	return &RetryHookRegistry{
		sessionHooks: make(map[string][]RetryHook),
	}
}

// RegisterGlobal adds a retry hook that runs for all sessions.
func (r *RetryHookRegistry) RegisterGlobal(hook RetryHook) {
	if hook == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.global = append(r.global, hook)
}

// RegisterSession adds a retry hook for a specific session.
func (r *RetryHookRegistry) RegisterSession(sessionID string, hook RetryHook) {
	if hook == nil || sessionID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionHooks[sessionID] = append(r.sessionHooks[sessionID], hook)
}

// GetHooks returns all retry hooks for a session (global + session-scoped).
func (r *RetryHookRegistry) GetHooks(sessionID string) []RetryHook {
	r.mu.RLock()
	defer r.mu.RUnlock()
	hooks := make([]RetryHook, 0, len(r.global)+len(r.sessionHooks[sessionID]))
	hooks = append(hooks, r.global...)
	if sessionHooks, ok := r.sessionHooks[sessionID]; ok {
		hooks = append(hooks, sessionHooks...)
	}
	return hooks
}

// ClearSession removes all session-scoped retry hooks for a session.
func (r *RetryHookRegistry) ClearSession(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessionHooks, sessionID)
}

// EvaluateRetryHooks runs all registered retry hooks and returns the final decision.
// The decision is determined by the most permissive hook: Retry > UseFallback > Fail.
// If no hooks are registered, the default is RetryDecisionFail.
func (r *RetryHookRegistry) EvaluateRetryHooks(ctx context.Context, retryCtx *RetryContext) (RetryDecision, error) {
	r.mu.RLock()
	hooks := make([]RetryHook, 0, len(r.global)+len(r.sessionHooks[retryCtx.SessionID]))
	hooks = append(hooks, r.global...)
	if sessionHooks, ok := r.sessionHooks[retryCtx.SessionID]; ok {
		hooks = append(hooks, sessionHooks...)
	}
	r.mu.RUnlock()

	if len(hooks) == 0 {
		return RetryDecisionFail, nil
	}

	var finalDecision = RetryDecisionFail
	var lastErr error

	for _, hook := range hooks {
		decision, err := hook(ctx, retryCtx)
		if err != nil {
			lastErr = err
			// An error from a hook is treated as a vote to fail
			continue
		}
		// More permissive decisions override less permissive ones
		if decision > finalDecision {
			finalDecision = decision
		}
	}

	return finalDecision, lastErr
}

// SessionRetryHook allows registering retry hooks on a session via context.
// Usage: ctx.session.hook("retry", hookFn) — attaches to session scope.
type SessionRetryHook struct {
	sessionID string
	registry  *RetryHookRegistry
}

func (h *SessionRetryHook) Hook(key string, hook RetryHook) {
	if h.registry != nil && key == "retry" {
		h.registry.RegisterSession(h.sessionID, hook)
	}
}

// NewSessionRetryHook creates a SessionRetryHook for a session.
func NewSessionRetryHook(sessionID string, registry *RetryHookRegistry) *SessionRetryHook {
	return &SessionRetryHook{sessionID: sessionID, registry: registry}
}