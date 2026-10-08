package runtime

import (
	"context"
	"time"

	"github.com/kothagpt/kotha/internal/llm/schema"
)

// Chatter is the model-call surface the loop needs (satisfied by route.Client).
type Chatter interface {
	Chat(ctx context.Context, messages []schema.Message, opts ...schema.Option) (schema.Completion, error)
}

// LoopConfig bounds the native agentic loop.
type LoopConfig struct {
	MaxTurns    int
	RequestOpts []schema.Option
}

func (c LoopConfig) maxTurns() int {
	if c.MaxTurns > 0 {
		return c.MaxTurns
	}
	return 10
}

// TurnResult is the settled outcome of a full model<->tools run.
type TurnResult struct {
	Messages   []schema.Message
	Completion schema.Completion
	Turns      int
	ToolCalls  int
}

// Runtime is the native LLM core: model turns interleaved with validated,
// bounded tool execution. Callers own persistence/streaming; the runtime
// owns the tool loop invariant.
type Runtime struct {
	Chat     Chatter
	Registry *Registry
	Exec     Executor
	Config   LoopConfig
}

func New(chat Chatter, reg *Registry, opts ...func(*Runtime)) *Runtime {
	rt := &Runtime{Chat: chat, Registry: reg, Exec: Executor{Registry: reg}, Config: LoopConfig{}}
	for _, fn := range opts {
		fn(rt)
	}
	if rt.Exec.Registry == nil {
		rt.Exec.Registry = reg
	}
	return rt
}

func WithMaxTurns(n int) func(*Runtime) {
	return func(r *Runtime) { r.Config.MaxTurns = n }
}

func WithRequestOpts(opts ...schema.Option) func(*Runtime) {
	return func(r *Runtime) { r.Config.RequestOpts = append(r.Config.RequestOpts, opts...) }
}

func WithExecOptions(o ExecOptions) func(*Runtime) {
	return func(r *Runtime) { r.Exec.Options = o }
}

// Run executes turns until the model answers without tool calls or the
// turn budget is exhausted. Tool results are appended as role=tool
// messages so the transcript stays canonical.
func (r *Runtime) Run(ctx context.Context, messages []schema.Message) (TurnResult, error) {
	transcript := append([]schema.Message(nil), messages...)
	toolDefs := r.Registry.Definitions()
	opts := append([]schema.Option{schema.WithTools(toolDefs...)}, r.Config.RequestOpts...)

	var last schema.Completion
	totalTools := 0
	turns := 0
	for ; turns < r.Config.maxTurns(); turns++ {
		if err := ctx.Err(); err != nil {
			return TurnResult{}, schema.NewError(schema.CodeCancelled, err)
		}
		comp, err := r.Chat.Chat(ctx, transcript, opts...)
		if err != nil {
			return TurnResult{}, err
		}
		last = comp
		if len(comp.ToolCalls) == 0 {
			transcript = append(transcript, completionMessage(comp))
			turns++
			break
		}
		transcript = append(transcript, completionMessage(comp))
		results := r.Exec.Execute(ctx, comp.ToolCalls)
		totalTools += len(results)
		transcript = append(transcript, schema.Message{
			ID: schema.NewMessageID(), Role: schema.RoleTool, Blocks: ToBlocks(results),
		})
	}
	return TurnResult{
		Messages:   transcript,
		Completion: last,
		Turns:      turns,
		ToolCalls:  totalTools,
	}, nil
}

func completionMessage(comp schema.Completion) schema.Message {
	blocks := []schema.ContentBlock{}
	if comp.Text != "" {
		blocks = append(blocks, schema.TextBlock(comp.Text))
	}
	for _, tc := range comp.ToolCalls {
		blocks = append(blocks, schema.ContentBlock{
			Type: schema.BlockToolCall, ID: tc.ID, Name: tc.Name, Input: tc.Input,
		})
	}
	return schema.Message{ID: schema.NewMessageID(), Role: schema.RoleAssistant, Blocks: blocks}
}

var _ = time.Second
