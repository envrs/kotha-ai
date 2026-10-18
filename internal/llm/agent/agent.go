package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kothagpt/kotha/internal/concurrency"
	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/fileutil"
	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/llm/prompt"
	"github.com/kothagpt/kotha/internal/llm/provider"
	"github.com/kothagpt/kotha/internal/llm/tools"
	"github.com/kothagpt/kotha/internal/logging"
	"github.com/kothagpt/kotha/internal/observability"
	"github.com/kothagpt/kotha/internal/message"
	"github.com/kothagpt/kotha/internal/permission"
	"github.com/kothagpt/kotha/internal/plugin"
	"github.com/kothagpt/kotha/internal/pubsub"
	"github.com/kothagpt/kotha/internal/session"
	"github.com/kothagpt/kotha/internal/toolcall"
)

// Common errors
var (
	ErrRequestCancelled = errors.New("request cancelled by user")
	ErrSessionBusy      = errors.New("session is currently processing another request")
	// ErrRetryRequested is a sentinel error indicating the retry hook requested a retry.
	ErrRetryRequested = errors.New("retry requested by hook")
)

type AgentEventType string

const (
	AgentEventTypeError     AgentEventType = "error"
	AgentEventTypeResponse  AgentEventType = "response"
	AgentEventTypeSummarize AgentEventType = "summarize"
)

type AgentEvent struct {
	Type    AgentEventType
	Message message.Message
	Error   error

	// When summarizing
	SessionID string
	Progress  string
	Done      bool
}

type Service interface {
	pubsub.Suscriber[AgentEvent]
	Model() models.Model
	Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error)
	Cancel(sessionID string)
	IsSessionBusy(sessionID string) bool
	IsBusy() bool
	Update(agentName config.AgentName, modelID models.ModelID) (models.Model, error)
	Summarize(ctx context.Context, sessionID string) error
	Meter() *observability.Meter
}

type agent struct {
	*pubsub.Broker[AgentEvent]
	sessions  session.Service
	messages  message.Service
	toolcalls toolcall.Service

	tools    []tools.BaseTool
	provider provider.Provider

	titleProvider     provider.Provider
	summarizeProvider provider.Provider

	activeRequests sync.Map

	configProvider config.ConfigProvider
	logger         logging.Logger
	meter          *observability.Meter

	// fileLock coordinates edits to the same file across concurrent
	// agent sessions. Nil when coordination is disabled.
	fileLock *FileLock

	// promptChain intercepts user prompts before admission (persistence
	// and LLM dispatch). Nil-safe: no interceptors when nil.
	promptChain *plugin.Chain

	// promptPipeline handles staged prompt preparation with idempotency,
	// interruptibility, and revert-on-failure.
	promptPipeline *plugin.PreparationPipeline

	// retryHookRegistry holds retry hooks for provider failure handling.
	// Nil-safe: no retry hooks when nil.
	retryHookRegistry *plugin.RetryHookRegistry
}

// AgentOption customizes an agent at construction.
type AgentOption func(*agent)

// WithPromptChain installs a prompt-interceptor chain. Every prompt is run
// through the chain before it is persisted or sent to the provider; a
// rejection produces an error event instead of starting a run.
func WithPromptChain(chain *plugin.Chain) AgentOption {
	return func(a *agent) { a.promptChain = chain }
}

// WithPromptPipeline installs a preparation pipeline for staged prompt
// preparation with idempotency, hooks, and revert-on-failure.
func WithPromptPipeline(pipeline *plugin.PreparationPipeline) AgentOption {
	return func(a *agent) { a.promptPipeline = pipeline }
}

// WithRetryHookRegistry installs a retry hook registry for provider failure
// handling. When a provider operation fails, the retry hooks are consulted
// to decide whether to retry, fail, or use a fallback.
func WithRetryHookRegistry(registry *plugin.RetryHookRegistry) AgentOption {
	return func(a *agent) { a.retryHookRegistry = registry }
}

func NewAgent(
	agentName config.AgentName,
	sessions session.Service,
	messages message.Service,
	toolcalls toolcall.Service,
	agentTools []tools.BaseTool,
	configProvider config.ConfigProvider,
	logger logging.Logger,
	opts ...AgentOption,
) (Service, error) {
	agentProvider, err := createAgentProvider(agentName, configProvider)
	if err != nil {
		return nil, err
	}
	var titleProvider provider.Provider
	// Only generate titles for the coder agent
	if agentName == config.AgentCoder {
		titleProvider, err = createAgentProvider(config.AgentTitle, configProvider)
		if err != nil {
			return nil, err
		}
	}
	var summarizeProvider provider.Provider
	if agentName == config.AgentCoder {
		summarizeProvider, err = createAgentProvider(config.AgentSummarizer, configProvider)
		if err != nil {
			return nil, err
		}
	}

	agent := &agent{
		Broker:            pubsub.NewBroker[AgentEvent](),
		provider:          agentProvider,
		messages:          messages,
		sessions:          sessions,
		toolcalls:         toolcalls,
		tools:             agentTools,
		titleProvider:     titleProvider,
		summarizeProvider: summarizeProvider,
		activeRequests:    sync.Map{},
		configProvider:    configProvider,
		logger:            logger,
		meter:             observability.NewMeter(),
		fileLock:          NewFileLock(30 * time.Second),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(agent)
		}
	}

	return agent, nil
}

func (a *agent) Model() models.Model {
	return a.provider.Model()
}

func (a *agent) Cancel(sessionID string) {
	if cancelFunc, exists := a.activeRequests.LoadAndDelete(sessionID); exists {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			a.logger.InfoPersist(fmt.Sprintf("Request cancellation initiated for session: %s", sessionID))
			cancel()
		}
	}

	// Also check for summarize requests
	if cancelFunc, exists := a.activeRequests.LoadAndDelete(sessionID + "-summarize"); exists {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			a.logger.InfoPersist(fmt.Sprintf("Summarize cancellation initiated for session: %s", sessionID))
			cancel()
		}
	}
}

func (a *agent) IsBusy() bool {
	busy := false
	a.activeRequests.Range(func(key, value interface{}) bool {
		if cancelFunc, ok := value.(context.CancelFunc); ok {
			if cancelFunc != nil {
				busy = true
				return false // Stop iterating
			}
		}
		return true // Continue iterating
	})
	return busy
}

func (a *agent) IsSessionBusy(sessionID string) bool {
	_, busy := a.activeRequests.Load(sessionID)
	return busy
}

func (a *agent) generateTitle(ctx context.Context, sessionID string, content string) error {
	if content == "" {
		return nil
	}
	if a.titleProvider == nil {
		return nil
	}
	session, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, tools.SessionIDContextKey, sessionID)
	parts := []message.ContentPart{message.TextContent{Text: content}}
	response, err := a.titleProvider.SendMessages(
		ctx,
		[]message.Message{
			{
				Role:  message.User,
				Parts: parts,
			},
		},
		make([]tools.BaseTool, 0),
	)
	if err != nil {
		return err
	}

	title := strings.TrimSpace(strings.ReplaceAll(response.Content, "\n", " "))
	if title == "" {
		return nil
	}

	session.Title = title
	_, err = a.sessions.Save(ctx, session)
	return err
}

func (a *agent) err(err error) AgentEvent {
	return AgentEvent{
		Type:  AgentEventTypeError,
		Error: err,
	}
}

func (a *agent) Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error) {
	if !a.provider.Model().SupportsAttachments && attachments != nil {
		attachments = nil
	}

	// Plugin admission gate: run prompt interceptors before anything is
	// persisted or dispatched. A rejection surfaces as an error event.
	content, attachments, admitErr := a.admitPrompt(ctx, sessionID, content, attachments)
	if admitErr != nil {
		events := make(chan AgentEvent, 1)
		events <- a.err(admitErr)
		close(events)
		return events, nil
	}

	if a.IsSessionBusy(sessionID) {
		return nil, ErrSessionBusy
	}

	events := make(chan AgentEvent, 1)
	genCtx, cancel := context.WithCancel(ctx)

	a.activeRequests.Store(sessionID, cancel)
	// Structured lifetime: single owned goroutine, always cleans up the
	// busy flag, cancel func, and channel — even on panic.
	// The channel is buffered (1) + sends are ctx-aware so a cancelled
	// consumer can never wedge the producer.
	go func() {
		defer cancel()
		defer a.activeRequests.Delete(sessionID)
		defer close(events)
		defer func() {
			if r := recover(); r != nil {
				a.logger.ErrorPersist(fmt.Sprintf("panic while running the agent: %v", r))
				concurrency.Send(genCtx, events, a.err(fmt.Errorf("panic while running the agent: %v", r)))
			}
		}()
		a.logger.Debug("Request started", "sessionID", sessionID)
		if a.meter != nil {
			a.meter.AgentRuns.Add(1)
		}
		var attachmentParts []message.ContentPart
		for _, attachment := range attachments {
			attachmentParts = append(attachmentParts, message.BinaryContent{Path: attachment.FilePath, MIMEType: attachment.MimeType, Data: attachment.Content})
		}
		result := a.processGeneration(genCtx, sessionID, content, attachmentParts)
		if result.Error != nil && !errors.Is(result.Error, ErrRequestCancelled) && !errors.Is(result.Error, context.Canceled) {
			a.logger.ErrorPersist(result.Error.Error())
		}
		if a.meter != nil {
			a.meter.Report()
		}
		a.logger.Debug("Request completed", "sessionID", sessionID)
		a.Publish(pubsub.CreatedEvent, result)
		concurrency.Send(genCtx, events, result)
	}()
	return events, nil
}

// admitPrompt runs the prompt-interceptor chain (if installed) against the
// user prompt. Interceptors may rewrite the content or attachments; a
// rejection (including an empty rewrite) is returned as an error.
func (a *agent) admitPrompt(ctx context.Context, sessionID, content string, attachments []message.Attachment) (string, []message.Attachment, error) {
	// Legacy chain support
	if a.promptChain != nil && a.promptChain.Len() > 0 {
		req := &plugin.PromptRequest{
			SessionID:   sessionID,
			Content:     content,
			Attachments: attachments,
		}
		if err := a.promptChain.Intercept(ctx, req); err != nil {
			return "", nil, err
		}
		if strings.TrimSpace(req.Content) == "" {
			return "", nil, &plugin.BlockedError{Interceptor: "agent", Reason: "prompt is empty after interceptors"}
		}
		return req.Content, req.Attachments, nil
	}

	// New preparation pipeline
	if a.promptPipeline != nil {
		// Build rich PromptRequest from legacy params
		req := &plugin.PromptRequest{
			SessionID:   sessionID,
			Text:        content,
			Delivery:    plugin.DeliveryDirect,
			AdmissionID: generateAdmissionID(sessionID, content),
		}
		for _, att := range attachments {
			req.Files = append(req.Files, plugin.FileDraft{
				ID:       generateFileDraftID(),
				Path:     att.FilePath,
				Name:     att.FileName,
				Size:     int64(len(att.Content)),
				MIMEType: att.MimeType,
				Content:  att.Content,
				Prepared: true,
			})
		}

		if err := a.promptPipeline.Prepare(ctx, req); err != nil {
			return "", nil, err
		}
		if req.AlreadyAdmitted {
			return "", nil, &plugin.BlockedError{Interceptor: "idempotency", Reason: "prompt already admitted"}
		}
		if req.Prepared {
			// Convert back to legacy format for processGeneration
			content = req.Text
			attachments = make([]message.Attachment, len(req.Files))
			for i, f := range req.Files {
				attachments[i] = message.Attachment{
					FilePath: f.Path,
					FileName: f.Name,
					MimeType: f.MIMEType,
					Content:  f.Content,
				}
			}
		}
		return content, attachments, nil
	}

	return content, attachments, nil
}

// generateAdmissionID creates a deterministic ID for idempotency.
func generateAdmissionID(sessionID, content string) string {
	// Simple hash-based ID; in production could use crypto
	h := 0
	for _, c := range sessionID + "|" + content {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return fmt.Sprintf("adm-%x", h)
}

// generateFileDraftID creates a unique draft ID.
func generateFileDraftID() string {
	return fmt.Sprintf("fd-%x", time.Now().UnixNano())
}

func (a *agent) processGeneration(ctx context.Context, sessionID, content string, attachmentParts []message.ContentPart) AgentEvent {
	cfg := a.configProvider.Get()
	// List existing messages; if none, start title generation asynchronously.
	msgs, err := a.messages.List(ctx, sessionID)
	if err != nil {
		return a.err(fmt.Errorf("failed to list messages: %w", err))
	}
	if len(msgs) == 0 {
		// Detached title generation: bounded 30s timeout, never blocks
		// the main generation loop, supervised with panic recovery.
		titleCtx, titleCancel := context.WithTimeout(context.Background(), 30*time.Second)
		_ = concurrency.GoDetached(titleCtx, func(ctx context.Context) error {
			defer titleCancel()
			defer func() {
				if r := recover(); r != nil {
					a.logger.ErrorPersist("panic while generating title")
				}
			}()
			if titleErr := a.generateTitle(ctx, sessionID, content); titleErr != nil {
				a.logger.ErrorPersist(fmt.Sprintf("failed to generate title: %v", titleErr))
			}
			return nil
		})
	}
	session, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return a.err(fmt.Errorf("failed to get session: %w", err))
	}
	if session.SummaryMessageID != "" {
		summaryMsgInex := -1
		for i, msg := range msgs {
			if msg.ID == session.SummaryMessageID {
				summaryMsgInex = i
				break
			}
		}
		if summaryMsgInex != -1 {
			msgs = msgs[summaryMsgInex:]
			msgs[0].Role = message.User
		}
	}

	userMsg, err := a.createUserMessage(ctx, sessionID, content, attachmentParts)
	if err != nil {
		return a.err(fmt.Errorf("failed to create user message: %w", err))
	}
	// Append the new user message to the conversation history.
	msgHistory := append(msgs, userMsg)

	for {
		// Check for cancellation before each iteration
		select {
		case <-ctx.Done():
			return a.err(ctx.Err())
		default:
			// Continue processing
		}
		agentMessage, toolResults, err := a.streamAndHandleEvents(ctx, sessionID, msgHistory)
		if err != nil {
			// Check if retry was requested
			if errors.Is(err, ErrRetryRequested) {
				a.logger.InfoPersist(fmt.Sprintf("Retrying generation for session %s", sessionID))
				// Restart the generation loop
				continue
			}
			if errors.Is(err, context.Canceled) {
				agentMessage.AddFinish(message.FinishReasonCanceled)
				_ = a.messages.Update(context.Background(), agentMessage)
				return a.err(ErrRequestCancelled)
			}
			return a.err(fmt.Errorf("failed to process events: %w", err))
		}
		if cfg.Debug {
			seqId := (len(msgHistory) + 1) / 2
			toolResultFilepath := a.logger.WriteToolResultsJson(sessionID, seqId, toolResults)
			a.logger.Info("Result", "message", agentMessage.FinishReason(), "toolResults", "{}", "filepath", toolResultFilepath)
		} else {
			a.logger.Info("Result", "message", agentMessage.FinishReason(), "toolResults", toolResults)
		}
		if (agentMessage.FinishReason() == message.FinishReasonToolUse) && toolResults != nil {
			// We are not done, we need to respond with the tool response
			msgHistory = append(msgHistory, agentMessage, *toolResults)
			continue
		}
		return AgentEvent{
			Type:    AgentEventTypeResponse,
			Message: agentMessage,
			Done:    true,
		}
	}
}

func (a *agent) createUserMessage(ctx context.Context, sessionID, content string, attachmentParts []message.ContentPart) (message.Message, error) {
	parts := []message.ContentPart{message.TextContent{Text: content}}
	parts = append(parts, attachmentParts...)
	return a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.User,
		Parts: parts,
	})
}

func (a *agent) streamAndHandleEvents(ctx context.Context, sessionID string, msgHistory []message.Message) (message.Message, *message.Message, error) {
	ctx = context.WithValue(ctx, tools.SessionIDContextKey, sessionID)
	eventChan := a.provider.StreamResponse(ctx, msgHistory, a.tools)

	assistantMsg, err := a.messages.Create(ctx, sessionID, message.CreateMessageParams{
		Role:  message.Assistant,
		Parts: []message.ContentPart{},
		Model: a.provider.Model().ID,
	})
	if err != nil {
		return assistantMsg, nil, fmt.Errorf("failed to create assistant message: %w", err)
	}

	// Add the session and message ID into the context if needed by tools.
	ctx = context.WithValue(ctx, tools.MessageIDContextKey, assistantMsg.ID)

	// Process each event in the stream.
	for event := range eventChan {
		if processErr := a.processEvent(ctx, sessionID, &assistantMsg, event); processErr != nil {
			// Check if retry was requested
			if errors.Is(processErr, ErrRetryRequested) {
				// Return a special error that signals retry to the caller
				return assistantMsg, nil, ErrRetryRequested
			}
			a.finishMessage(ctx, &assistantMsg, message.FinishReasonCanceled)
			return assistantMsg, nil, processErr
		}
		if ctx.Err() != nil {
			a.finishMessage(context.Background(), &assistantMsg, message.FinishReasonCanceled)
			return assistantMsg, nil, ctx.Err()
		}
	}

	toolResults := make([]message.ToolResult, len(assistantMsg.ToolCalls()))
	toolCalls := assistantMsg.ToolCalls()
	for i, toolCall := range toolCalls {
		select {
		case <-ctx.Done():
			a.finishMessage(context.Background(), &assistantMsg, message.FinishReasonCanceled)
			// Make all future tool calls cancelled
			for j := i; j < len(toolCalls); j++ {
				toolResults[j] = message.ToolResult{
					ToolCallID: toolCalls[j].ID,
					Content:    "Tool execution canceled by user",
					IsError:    true,
				}
			}
			goto out
		default:
			// Continue processing
			var tool tools.BaseTool
			for _, availableTool := range a.tools {
				if availableTool.Info().Name == toolCall.Name {
					tool = availableTool
					break
				}
				// Monkey patch for Copilot Sonnet-4 tool repetition obfuscation
				// if strings.HasPrefix(toolCall.Name, availableTool.Info().Name) &&
				// 	strings.HasPrefix(toolCall.Name, availableTool.Info().Name+availableTool.Info().Name) {
				// 	tool = availableTool
				// 	break
				// }
			}

			// Tool not found
			if tool == nil {
				toolResults[i] = message.ToolResult{
					ToolCallID: toolCall.ID,
					Content:    fmt.Sprintf("Tool not found: %s", toolCall.Name),
					IsError:    true,
				}
				continue
			}
			// Coordinate edits to the same file across concurrent agent
			// sessions. Write/edit tools expose a file path; non-file tools
			// are unaffected.
			filePath := extractFilePath(toolCall.Input)
			if filePath != "" && a.fileLock != nil {
				if lockErr := a.fileLock.Acquire(ctx, sessionID, filePath); lockErr != nil {
					toolResults[i] = message.ToolResult{
						ToolCallID: toolCall.ID,
						Content:    lockErr.Error(),
						IsError:    true,
					}
					continue
				}
			}
			start := time.Now()
			toolResult, toolErr := tool.Run(ctx, tools.ToolCall{
				ID:    toolCall.ID,
				Name:  toolCall.Name,
				Input: toolCall.Input,
			})
			if filePath != "" && a.fileLock != nil {
				a.fileLock.Release(sessionID, filePath)
			}
			// Record the tool invocation for history and cost auditing.
			a.recordToolCall(ctx, sessionID, assistantMsg.ID, toolCall, toolResult, toolErr, time.Since(start))
			if toolErr != nil {
				if errors.Is(toolErr, permission.ErrorPermissionDenied) {
					toolResults[i] = message.ToolResult{
						ToolCallID: toolCall.ID,
						Content:    "Permission denied",
						IsError:    true,
					}
					for j := i + 1; j < len(toolCalls); j++ {
						toolResults[j] = message.ToolResult{
							ToolCallID: toolCalls[j].ID,
							Content:    "Tool execution canceled by user",
							IsError:    true,
						}
					}
					a.finishMessage(ctx, &assistantMsg, message.FinishReasonPermissionDenied)
					break
				}
			}
			toolResults[i] = message.ToolResult{
				ToolCallID: toolCall.ID,
				Content:    toolResult.Content,
				Metadata:   toolResult.Metadata,
				IsError:    toolResult.IsError,
			}
		}
	}
out:
	if len(toolResults) == 0 {
		return assistantMsg, nil, nil
	}
	parts := make([]message.ContentPart, 0)
	for _, tr := range toolResults {
		parts = append(parts, tr)
	}
	msg, err := a.messages.Create(context.Background(), assistantMsg.SessionID, message.CreateMessageParams{
		Role:  message.Tool,
		Parts: parts,
	})
	if err != nil {
		return assistantMsg, nil, fmt.Errorf("failed to create cancelled tool message: %w", err)
	}

	return assistantMsg, &msg, err
}

func (a *agent) finishMessage(ctx context.Context, msg *message.Message, finishReson message.FinishReason) {
	msg.AddFinish(finishReson)
	_ = a.messages.Update(ctx, *msg)
}

// recordToolCall persists a tool invocation for history and cost auditing.
// Failures here are non-fatal: a lost audit record must never break the agent.
func (a *agent) recordToolCall(ctx context.Context, sessionID, messageID string, call message.ToolCall, result tools.ToolResponse, runErr error, duration time.Duration) {
	if a.toolcalls == nil {
		return
	}
	if a.meter != nil {
		a.meter.ToolCalls.Add(1)
		if runErr != nil || result.IsError {
			a.meter.ToolErrors.Add(1)
		}
		a.meter.Latency.Record(float64(duration.Milliseconds()))
	}
	isError := runErr != nil || result.IsError
	output := result.Content
	if runErr != nil {
		output = runErr.Error()
	}
	_, _ = a.toolcalls.Create(ctx, toolcall.ToolCall{
		SessionID:  sessionID,
		MessageID:  messageID,
		ToolCallID: call.ID,
		ToolName:   call.Name,
		Input:      call.Input,
		Output:     output,
		IsError:    isError,
		DurationMs: duration.Milliseconds(),
	})
}

func (a *agent) processEvent(ctx context.Context, sessionID string, assistantMsg *message.Message, event provider.ProviderEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Continue processing.
	}

	switch event.Type {
	case provider.EventThinkingDelta:
		assistantMsg.AppendReasoningContent(event.Content)
		return a.messages.Update(ctx, *assistantMsg)
	case provider.EventContentDelta:
		assistantMsg.AppendContent(event.Content)
		return a.messages.Update(ctx, *assistantMsg)
	case provider.EventToolUseStart:
		assistantMsg.AddToolCall(*event.ToolCall)
		return a.messages.Update(ctx, *assistantMsg)
	// TODO: see how to handle this
	// case provider.EventToolUseDelta:
	// 	tm := time.Unix(assistantMsg.UpdatedAt, 0)
	// 	assistantMsg.AppendToolCallInput(event.ToolCall.ID, event.ToolCall.Input)
	// 	if time.Since(tm) > 1000*time.Millisecond {
	// 		err := a.messages.Update(ctx, *assistantMsg)
	// 		assistantMsg.UpdatedAt = time.Now().Unix()
	// 		return err
	// 	}
	case provider.EventToolUseStop:
		assistantMsg.FinishToolCall(event.ToolCall.ID)
		return a.messages.Update(ctx, *assistantMsg)
	case provider.EventError:
		if errors.Is(event.Error, context.Canceled) {
			a.logger.InfoPersist(fmt.Sprintf("Event processing canceled for session: %s", sessionID))
			return context.Canceled
		}

		// Check retry hooks for provider failure
		if a.retryHookRegistry != nil {
			retryCtx := &plugin.RetryContext{
				SessionID:    sessionID,
				Attempt:      1, // This is the first agent-level retry attempt
				MaxAttempts:  1, // Only one agent-level retry by default
				Error:        event.Error,
				ProviderName: string(a.provider.Model().Provider),
				ModelID:      string(a.provider.Model().ID),
				IsStream:     true,
			}

			decision, err := a.retryHookRegistry.EvaluateRetryHooks(ctx, retryCtx)
			if err != nil {
				a.logger.ErrorPersist(fmt.Sprintf("Retry hook error: %v", err))
			}

			switch decision {
			case plugin.RetryDecisionRetry:
				a.logger.InfoPersist(fmt.Sprintf("Retrying request for session %s per retry hook", sessionID))
				// Return special error to signal retry to the caller
				return ErrRetryRequested
			case plugin.RetryDecisionUseFallback:
				a.logger.InfoPersist(fmt.Sprintf("Using fallback for session %s per retry hook", sessionID))
				// Could implement fallback logic here
			case plugin.RetryDecisionFail:
				fallthrough
			default:
				// Default: fail
			}
		}

		a.logger.ErrorPersist(event.Error.Error())
		return event.Error
	case provider.EventComplete:
		assistantMsg.SetToolCalls(event.Response.ToolCalls)
		assistantMsg.AddFinish(event.Response.FinishReason)
		if err := a.messages.Update(ctx, *assistantMsg); err != nil {
			return fmt.Errorf("failed to update message: %w", err)
		}
		return a.TrackUsage(ctx, sessionID, a.provider.Model(), event.Response.Usage)
	}

	return nil
}

// TrackUsage records token usage and cost for a session.
func (a *agent) TrackUsage(ctx context.Context, sessionID string, model models.Model, usage provider.TokenUsage) error {
	sess, err := a.sessions.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	cost := model.CostPer1MInCached/1e6*float64(usage.CacheCreationTokens) +
		model.CostPer1MOutCached/1e6*float64(usage.CacheReadTokens) +
		model.CostPer1MIn/1e6*float64(usage.InputTokens) +
		model.CostPer1MOut/1e6*float64(usage.OutputTokens)

	sess.Cost += cost
	sess.CompletionTokens = usage.OutputTokens + usage.CacheReadTokens
	sess.PromptTokens = usage.InputTokens + usage.CacheCreationTokens

	_, err = a.sessions.Save(ctx, sess)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	return nil
}

func (a *agent) Update(agentName config.AgentName, modelID models.ModelID) (models.Model, error) {
	if a.IsBusy() {
		return models.Model{}, fmt.Errorf("cannot change model while processing requests")
	}

	if err := a.configProvider.UpdateAgentModel(agentName, modelID); err != nil {
		return models.Model{}, fmt.Errorf("failed to update config: %w", err)
	}

	provider, err := createAgentProvider(agentName, a.configProvider)
	if err != nil {
		return models.Model{}, fmt.Errorf("failed to create provider for model %s: %w", modelID, err)
	}

	a.provider = provider

	return a.provider.Model(), nil
}

func (a *agent) Summarize(ctx context.Context, sessionID string) error {
	if a.summarizeProvider == nil {
		return fmt.Errorf("summarize provider not available")
	}

	// Check if session is busy
	if a.IsSessionBusy(sessionID) {
		return ErrSessionBusy
	}

	// Create a new context with cancellation
	summarizeCtx, cancel := context.WithCancel(ctx)

	// Store the cancel function in activeRequests to allow cancellation
	a.activeRequests.Store(sessionID+"-summarize", cancel)

	go func() {
		defer a.activeRequests.Delete(sessionID + "-summarize")
		defer cancel()
		event := AgentEvent{
			Type:     AgentEventTypeSummarize,
			Progress: "Starting summarization...",
		}

		a.Publish(pubsub.CreatedEvent, event)
		// Get all messages from the session
		msgs, err := a.messages.List(summarizeCtx, sessionID)
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to list messages: %w", err),
				Done:  true,
			}
			a.Publish(pubsub.CreatedEvent, event)
			return
		}
		summarizeCtx = context.WithValue(summarizeCtx, tools.SessionIDContextKey, sessionID)

		if len(msgs) == 0 {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("no messages to summarize"),
				Done:  true,
			}
			a.Publish(pubsub.CreatedEvent, event)
			return
		}

		event = AgentEvent{
			Type:     AgentEventTypeSummarize,
			Progress: "Analyzing conversation...",
		}
		a.Publish(pubsub.CreatedEvent, event)

		// Add a system message to guide the summarization
		summarizePrompt := "Provide a detailed but concise summary of our conversation above. Focus on information that would be helpful for continuing the conversation, including what we did, what we're doing, which files we're working on, and what we're going to do next."

		// Create a new message with the summarize prompt
		promptMsg := message.Message{
			Role:  message.User,
			Parts: []message.ContentPart{message.TextContent{Text: summarizePrompt}},
		}

		// Append the prompt to the messages
		msgsWithPrompt := append(msgs, promptMsg)

		event = AgentEvent{
			Type:     AgentEventTypeSummarize,
			Progress: "Generating summary...",
		}

		a.Publish(pubsub.CreatedEvent, event)

		// Send the messages to the summarize provider
		response, err := a.summarizeProvider.SendMessages(
			summarizeCtx,
			msgsWithPrompt,
			make([]tools.BaseTool, 0),
		)
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to summarize: %w", err),
				Done:  true,
			}
			a.Publish(pubsub.CreatedEvent, event)
			return
		}

		summary := strings.TrimSpace(response.Content)
		if summary == "" {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("empty summary returned"),
				Done:  true,
			}
			a.Publish(pubsub.CreatedEvent, event)
			return
		}
		event = AgentEvent{
			Type:     AgentEventTypeSummarize,
			Progress: "Creating new session...",
		}

		a.Publish(pubsub.CreatedEvent, event)
		oldSession, err := a.sessions.Get(summarizeCtx, sessionID)
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to get session: %w", err),
				Done:  true,
			}

			a.Publish(pubsub.CreatedEvent, event)
			return
		}
		// Create a message in the new session with the summary
		msg, err := a.messages.Create(summarizeCtx, oldSession.ID, message.CreateMessageParams{
			Role: message.Assistant,
			Parts: []message.ContentPart{
				message.TextContent{Text: summary},
				message.Finish{
					Reason: message.FinishReasonEndTurn,
					Time:   time.Now().Unix(),
				},
			},
			Model: a.summarizeProvider.Model().ID,
		})
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to create summary message: %w", err),
				Done:  true,
			}

			a.Publish(pubsub.CreatedEvent, event)
			return
		}
		oldSession.SummaryMessageID = msg.ID
		oldSession.CompletionTokens = response.Usage.OutputTokens
		oldSession.PromptTokens = 0
		model := a.summarizeProvider.Model()
		usage := response.Usage
		cost := model.CostPer1MInCached/1e6*float64(usage.CacheCreationTokens) +
			model.CostPer1MOutCached/1e6*float64(usage.CacheReadTokens) +
			model.CostPer1MIn/1e6*float64(usage.InputTokens) +
			model.CostPer1MOut/1e6*float64(usage.OutputTokens)
		oldSession.Cost += cost
		_, err = a.sessions.Save(summarizeCtx, oldSession)
		if err != nil {
			event = AgentEvent{
				Type:  AgentEventTypeError,
				Error: fmt.Errorf("failed to save session: %w", err),
				Done:  true,
			}
			a.Publish(pubsub.CreatedEvent, event)
		}

		event = AgentEvent{
			Type:      AgentEventTypeSummarize,
			SessionID: oldSession.ID,
			Progress:  "Summary complete",
			Done:      true,
		}
		a.Publish(pubsub.CreatedEvent, event)
		// Send final success event with the new session ID
	}()

	return nil
}

func createAgentProvider(agentName config.AgentName, configProvider config.ConfigProvider) (provider.Provider, error) {
	cfg := configProvider.Get()
	agentConfig, ok := cfg.Agents[agentName]
	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentName)
	}
	model, ok := models.SupportedModels[agentConfig.Model]
	if !ok {
		return nil, fmt.Errorf("model %s not supported", agentConfig.Model)
	}

	providerCfg, ok := cfg.Providers[model.Provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not supported", model.Provider)
	}
	if providerCfg.Disabled {
		return nil, fmt.Errorf("provider %s is not enabled", model.Provider)
	}
	maxTokens := model.DefaultMaxTokens
	if agentConfig.MaxTokens > 0 {
		maxTokens = agentConfig.MaxTokens
	}
	opts := []provider.ProviderClientOption{
		provider.WithAPIKey(providerCfg.APIKey),
		provider.WithModel(model),
		provider.WithSystemMessage(prompt.GetAgentPrompt(agentName, model.Provider)),
		provider.WithMaxTokens(maxTokens),
	}
	if model.Provider == models.ProviderOpenAI || model.Provider == models.ProviderLocal && model.CanReason {
		opts = append(
			opts,
			provider.WithOpenAIOptions(
				provider.WithReasoningEffort(agentConfig.ReasoningEffort),
			),
		)
	} else if model.Provider == models.ProviderAnthropic && model.CanReason && agentName == config.AgentCoder {
		opts = append(
			opts,
			provider.WithAnthropicOptions(
				provider.WithAnthropicShouldThinkFn(provider.DefaultShouldThinkFn),
			),
		)
	}
	agentProvider, err := provider.NewProvider(
		model.Provider,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create provider: %v", err)
	}

	return agentProvider, nil
}

func (a *agent) Meter() *observability.Meter {
	return a.meter
}

// extractFilePath pulls a file_path field out of a tool call's JSON input.
// Returns "" for tools that don't operate on files.
// Expands tilde (~) in the path so lock keys match the expanded paths used by tools.
func extractFilePath(input string) string {
	if input == "" {
		return ""
	}
	var parsed struct {
		FilePath string `json:"file_path"`
	}
	if err := json.Unmarshal([]byte(input), &parsed); err != nil {
		return ""
	}
	return fileutil.ExpandTilde(parsed.FilePath)
}
