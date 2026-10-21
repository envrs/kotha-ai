package testllm

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/kothagpt/kotha/internal/concurrency"
	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/llm/provider"
	"github.com/kothagpt/kotha/internal/llm/tools"
	"github.com/kothagpt/kotha/internal/message"
)

// RequestMatcher is a function that determines if a scripted response matches a request.
// Return true to use the associated response.
type RequestMatcher func(messages []message.Message, tools []tools.BaseTool) bool

// ScriptedResponse represents a pre-defined response that can be matched to requests.
type ScriptedResponse struct {
	Matcher  RequestMatcher
	Response *provider.ProviderResponse
	Error    error
}

// TestLLMController provides first-class control over scripted LLM responses for tests.
// It manages a queue of scripted responses, request matchers, and call recording.
type TestLLMController struct {
	mu sync.Mutex

	// Queue of scripted responses that are matched against requests.
	scripts []ScriptedResponse

	// Fallback response used when no script matches.
	fallback *provider.ProviderResponse
	fallbackErr error

	// Stream-specific error (applies to all streams).
	streamErr error

	// Recorded calls for assertions.
	calls []MockCall

	// Whether to advance to the next script after a match (consumes the script).
	// When false, the script stays in the queue and can be reused.
	consumeOnMatch bool
}

// MockCall records a single call to the LLM for later inspection.
type MockCall struct {
	Messages []message.Message
	Tools    []tools.BaseTool
	// The script that matched this call (nil if fallback was used).
	MatchedScript *ScriptedResponse
	// The response returned.
	Response *provider.ProviderResponse
	Err      error
}

// NewTestLLMController creates a new test controller with default settings.
// By default, scripts are consumed on match (advancing behavior).
func NewTestLLMController() *TestLLMController {
	return &TestLLMController{
		consumeOnMatch: true,
	}
}

// WithConsumeOnMatch controls whether matched scripts are removed from the queue.
// Default is true (advancing behavior). Set to false for reusable scripts.
func (c *TestLLMController) WithConsumeOnMatch(consume bool) *TestLLMController {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consumeOnMatch = consume
	return c
}

// AddScript adds a scripted response with an optional matcher.
// If matcher is nil, the script matches any request (fallback-like).
// Scripts are evaluated in order; the first match wins.
func (c *TestLLMController) AddScript(matcher RequestMatcher, resp *provider.ProviderResponse, err error) *TestLLMController {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scripts = append(c.scripts, ScriptedResponse{
		Matcher:  matcher,
		Response: resp,
		Error:    err,
	})
	return c
}

// AddResponse adds a script that matches any request (catch-all).
// Useful for simple sequential responses.
func (c *TestLLMController) AddResponse(resp *provider.ProviderResponse) *TestLLMController {
	return c.AddScript(nil, resp, nil)
}

// AddError adds a script that returns an error for any request.
func (c *TestLLMController) AddError(err error) *TestLLMController {
	return c.AddScript(nil, nil, err)
}

// SetFallback sets the default response when no script matches.
func (c *TestLLMController) SetFallback(resp *provider.ProviderResponse, err error) *TestLLMController {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fallback = resp
	c.fallbackErr = err
	return c
}

// SetStreamError sets an error that will be returned for all stream calls.
func (c *TestLLMController) SetStreamError(err error) *TestLLMController {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.streamErr = err
	return c
}

// Calls returns a copy of all recorded calls for assertion.
func (c *TestLLMController) Calls() []MockCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	calls := make([]MockCall, len(c.calls))
	copy(calls, c.calls)
	return calls
}

// LastCall returns the most recent call, or nil if none.
func (c *TestLLMController) LastCall() *MockCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.calls) == 0 {
		return nil
	}
	call := c.calls[len(c.calls)-1]
	return &call
}

// CallCount returns the number of recorded calls.
func (c *TestLLMController) CallCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.calls)
}

// Reset clears all scripts, calls, and errors.
func (c *TestLLMController) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scripts = nil
	c.calls = nil
	c.fallback = nil
	c.fallbackErr = nil
	c.streamErr = nil
}

// ExpectCall verifies that exactly one call was made and returns it.
// Panics if the number of calls is not exactly one.
func (c *TestLLMController) ExpectCall() MockCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.calls) != 1 {
		panic(fmt.Sprintf("expected exactly 1 call, got %d", len(c.calls)))
	}
	return c.calls[0]
}

// ExpectCalls verifies the exact number of calls and returns them.
// Panics if the count doesn't match.
func (c *TestLLMController) ExpectCalls(n int) []MockCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.calls) != n {
		panic(fmt.Sprintf("expected %d calls, got %d", n, len(c.calls)))
	}
	calls := make([]MockCall, len(c.calls))
	copy(calls, c.calls)
	return calls
}

// ExpectNoCalls panics if any calls were made.
func (c *TestLLMController) ExpectNoCalls() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.calls) > 0 {
		panic(fmt.Sprintf("expected no calls, got %d", len(c.calls)))
	}
}

// ExpectCallMatching verifies that at least one call matches the predicate.
// Returns the first matching call.
func (c *TestLLMController) ExpectCallMatching(pred func(MockCall) bool) MockCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, call := range c.calls {
		if pred(call) {
			return call
		}
	}
	panic("no call matched predicate")
}

// BuildProvider creates a provider.Provider backed by this controller.
// This is the primary way to inject the TestLLM into the normal provider chain.
func (c *TestLLMController) BuildProvider(modelID string) provider.Provider {
	return &testLLMProvider{
		controller: c,
		modelID:    models.ModelID(modelID),
	}
}

// BuildProviderWithOptions creates a provider with additional provider options.
// The model from opts will override the modelID if set.
func (c *TestLLMController) BuildProviderWithOptions(opts ...provider.ProviderClientOption) (provider.Provider, error) {
	opts = append(opts, provider.WithMockOptions(
		provider.WithMockController(c),
	))
	// We need a dummy provider name; the mock client will be used.
	return provider.NewProvider(models.ProviderMock, opts...)
}

// testLLMProvider wraps a TestLLMController to implement provider.Provider.
type testLLMProvider struct {
	controller *TestLLMController
	modelID    models.ModelID
}

func (p *testLLMProvider) SendMessages(ctx context.Context, messages []message.Message, tools []tools.BaseTool) (*provider.ProviderResponse, error) {
	// For non-streaming, we delegate to the stream and collect the final response.
	ch := p.controller.Stream(ctx, messages, tools)
	var final *provider.ProviderResponse
	var finalErr error
	for ev := range ch {
		if ev.Type == provider.EventError && ev.Error != nil {
			return nil, ev.Error
		}
		if ev.Type == provider.EventComplete && ev.Response != nil {
			final = ev.Response
			finalErr = ev.Error
		}
	}
	return final, finalErr
}

func (p *testLLMProvider) StreamResponse(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan provider.ProviderEvent {
	return p.controller.Stream(ctx, messages, tools)
}

func (p *testLLMProvider) Model() models.Model {
	return models.Model{ID: p.modelID}
}

// ---- Internal implementation ----

// Stream implements the TestController interface.
func (c *TestLLMController) Stream(ctx context.Context, messages []message.Message, tools []tools.BaseTool) <-chan provider.ProviderEvent {
	ch := make(chan provider.ProviderEvent, 8)

	// Capture the call
	call := MockCall{
		Messages: messages,
		Tools:    tools,
	}

	c.mu.Lock()
	c.calls = append(c.calls, call)
	c.mu.Unlock()

	go func() {
		defer close(ch)

		// Check for stream error
		c.mu.Lock()
		streamErr := c.streamErr
		c.mu.Unlock()

		if streamErr != nil {
			concurrency.Send(ctx, ch, provider.ProviderEvent{Type: provider.EventError, Error: streamErr})
			return
		}

		// Find matching script
		var matchedScript *ScriptedResponse
		var resp *provider.ProviderResponse
		var err error

		c.mu.Lock()

		// Find first matching script
		for i, script := range c.scripts {
			if script.Matcher == nil || script.Matcher(messages, tools) {
				matchedScript = &c.scripts[i]
				resp = script.Response
				err = script.Error

				// Consume the script if configured
				if c.consumeOnMatch {
					c.scripts = append(c.scripts[:i], c.scripts[i+1:]...)
				}
				break
			}
		}

		// Fallback if no script matched
		if matchedScript == nil {
			resp = c.fallback
			err = c.fallbackErr
		}

		c.mu.Unlock()

		// Record the call details
		c.mu.Lock()
		if len(c.calls) > 0 {
			lastIdx := len(c.calls) - 1
			c.calls[lastIdx].MatchedScript = matchedScript
			c.calls[lastIdx].Response = resp
			c.calls[lastIdx].Err = err
		}
		c.mu.Unlock()

		if err != nil {
			concurrency.Send(ctx, ch, provider.ProviderEvent{Type: provider.EventError, Error: err})
			return
		}

		if resp == nil {
			resp = &provider.ProviderResponse{FinishReason: message.FinishReasonEndTurn}
		}

		if resp.Content != "" {
			concurrency.Send(ctx, ch, provider.ProviderEvent{Type: provider.EventContentDelta, Content: resp.Content})
		}
		for _, tc := range resp.ToolCalls {
			tcCopy := tc
			concurrency.Send(ctx, ch, provider.ProviderEvent{Type: provider.EventToolUseStart, ToolCall: &tcCopy})
		}
		concurrency.Send(ctx, ch, provider.ProviderEvent{Type: provider.EventComplete, Response: resp})
	}()

	return ch
}

// Matcher helpers for common patterns.

// MatchContent returns a matcher that matches when the last user message contains the substring.
func MatchContent(substring string) RequestMatcher {
	return func(messages []message.Message, _ []tools.BaseTool) bool {
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == message.User {
				content := messages[i].Content().String()
				if len(content) > 0 && contains(content, substring) {
					return true
				}
			}
		}
		return false
	}
}

// MatchToolCall returns a matcher that matches when the request includes a tool call with the given name.
func MatchToolCall(name string) RequestMatcher {
	return func(_ []message.Message, tools []tools.BaseTool) bool {
		for _, tool := range tools {
			if tool.Info().Name == name {
				return true
			}
		}
		return false
	}
}

// MatchAnyToolCall matches if any tool call is present in the request.
func MatchAnyToolCall() RequestMatcher {
	return func(_ []message.Message, tools []tools.BaseTool) bool {
		return len(tools) > 0
	}
}

// MatchRole matches if the last message has the given role.
func MatchRole(role message.MessageRole) RequestMatcher {
	return func(messages []message.Message, _ []tools.BaseTool) bool {
		if len(messages) == 0 {
			return false
		}
		return messages[len(messages)-1].Role == role
	}
}

// MatchSystemPrompt matches if the first message is a system message containing the substring.
func MatchSystemPrompt(substring string) RequestMatcher {
	return func(messages []message.Message, _ []tools.BaseTool) bool {
		if len(messages) > 0 && messages[0].Role == message.System {
			content := messages[0].Content().String()
			return contains(content, substring)
		}
		return false
	}
}

// MatchMessageCount matches if the conversation has exactly n messages.
func MatchMessageCount(n int) RequestMatcher {
	return func(messages []message.Message, _ []tools.BaseTool) bool {
		return len(messages) == n
	}
}

// MatchAll matches any request (always true).
func MatchAll() RequestMatcher {
	return func(_ []message.Message, _ []tools.BaseTool) bool {
		return true
	}
}

// contains is a case-insensitive substring check.
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	sLower := toLower(s)
	subLower := toLower(substr)
	for i := 0; i <= len(sLower)-len(subLower); i++ {
		if sLower[i:i+len(subLower)] == subLower {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	// Simple ASCII lowercasing
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		} else {
			b[i] = c
		}
	}
	return string(b)
}

// Re-export types from provider for convenience.
var (
	ErrNoResponse = errors.New("no response available")
)