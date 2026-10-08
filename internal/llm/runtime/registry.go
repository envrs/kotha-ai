package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/kothagpt/kotha/internal/llm/schema"
	legtools "github.com/kothagpt/kotha/internal/llm/tools"
)

// Registry is the native tool catalog: lookup, definitions, legacy bridge.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

func (r *Registry) Register(t Tool) error {
	def := t.Definition()
	if def.Name == "" {
		return schema.NewError(schema.CodeInvalidRequest, schema.ErrInvalidRequest)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[def.Name]; exists {
		return schema.NewError(schema.CodeInvalidRequest, fmt.Errorf("tool %q already registered", def.Name))
	}
	r.tools[def.Name] = t
	return nil
}

func (r *Registry) MustRegister(t Tool) *Registry {
	if err := r.Register(t); err != nil {
		panic(err)
	}
	return r
}

func (r *Registry) Lookup(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// Definitions returns schema tool defs for model requests, sorted by name.
func (r *Registry) Definitions() []schema.ToolDef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]schema.ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t.Definition())
	}
	// Deterministic order for prompts/snapshots.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Name < out[j-1].Name; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// legacyAdapter bridges tools.BaseTool into the native Tool interface.
type legacyAdapter struct {
	base legtools.BaseTool
}

func (a legacyAdapter) Definition() schema.ToolDef {
	info := a.base.Info()
	return schema.ToolDef{
		Name:        info.Name,
		Description: info.Description,
		Parameters:  info.Parameters,
		Required:    info.Required,
	}
}

func (a legacyAdapter) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	resp, err := a.base.Run(ctx, legtools.ToolCall{Name: a.base.Info().Name, Input: string(input)})
	if err != nil {
		return "", err
	}
	if resp.IsError {
		return "", fmt.Errorf("%s", resp.Content)
	}
	return resp.Content, nil
}

// RegisterLegacy adapts and registers a legacy tools.BaseTool.
func (r *Registry) RegisterLegacy(t legtools.BaseTool) error {
	return r.Register(legacyAdapter{base: t})
}
