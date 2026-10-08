package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kothagpt/kotha/internal/llm/schema"
)

// ExecOptions bounds a single tool-call batch.
type ExecOptions struct {
	Timeout       time.Duration
	MaxConcurrent int
}

func (o ExecOptions) timeout() time.Duration {
	if o.Timeout > 0 {
		return o.Timeout
	}
	return 2 * time.Minute
}

func (o ExecOptions) concurrency() int {
	if o.MaxConcurrent > 0 {
		return o.MaxConcurrent
	}
	return 4
}

// ExecResult is the settled outcome of one tool call.
type ExecResult struct {
	Call     schema.ToolCall
	Output   string
	Err      error
	Duration time.Duration
}

// Executor runs tool calls against a Registry with timeout, concurrency
// cap, input validation, and panic recovery.
type Executor struct {
	Registry *Registry
	Options  ExecOptions
}

func (e Executor) Execute(ctx context.Context, calls []schema.ToolCall) []ExecResult {
	results := make([]ExecResult, len(calls))
	sem := make(chan struct{}, e.Options.concurrency())
	var wg sync.WaitGroup
	for i, call := range calls {
		wg.Add(1)
		go func(i int, call schema.ToolCall) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[i] = ExecResult{Call: call, Err: ctx.Err()}
				return
			}
			results[i] = e.executeOne(ctx, call)
		}(i, call)
	}
	wg.Wait()
	return results
}

func (e Executor) executeOne(ctx context.Context, call schema.ToolCall) (res ExecResult) {
	start := time.Now()
	res.Call = call
	defer func() {
		res.Duration = time.Since(start)
		if r := recover(); r != nil {
			res.Err = schema.NewError(schema.CodeProvider, fmt.Errorf("tool %q panicked: %v", call.Name, r))
		}
	}()
	tool, ok := e.Registry.Lookup(call.Name)
	if !ok {
		res.Err = schema.NewError(schema.CodeNotFound, fmt.Errorf("unknown tool %q", call.Name))
		return res
	}
	raw := json.RawMessage(call.Input)
	if err := ValidateInput(tool.Definition(), raw); err != nil {
		res.Err = err
		return res
	}
	inner, cancel := context.WithTimeout(ctx, e.Options.timeout())
	defer cancel()
	out, err := tool.Execute(inner, raw)
	if err != nil {
		res.Err = err
		return res
	}
	res.Output = out
	return res
}

// ToBlocks converts settled results into tool_result content blocks.
func ToBlocks(results []ExecResult) []schema.ContentBlock {
	blocks := make([]schema.ContentBlock, 0, len(results))
	for _, r := range results {
		content := r.Output
		isErr := r.Err != nil
		if isErr {
			content = r.Err.Error()
		}
		blocks = append(blocks, schema.ContentBlock{
			Type:       schema.BlockToolResult,
			ToolCallID: r.Call.ID,
			Name:       r.Call.Name,
			Content:    content,
			IsError:    isErr,
		})
	}
	return blocks
}
