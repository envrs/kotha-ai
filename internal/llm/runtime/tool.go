package runtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kothagpt/kotha/internal/llm/schema"
)

// Tool is the native executable unit. Implementations are pure Go;
// adapters bridge legacy tools.BaseTool into this interface.
type Tool interface {
	Definition() schema.ToolDef
	Execute(ctx context.Context, input json.RawMessage) (string, error)
}

// ToolFunc adapts a plain function into a Tool.
type ToolFunc struct {
	Def schema.ToolDef
	Fn  func(ctx context.Context, input json.RawMessage) (string, error)
}

func (t ToolFunc) Definition() schema.ToolDef { return t.Def }

func (t ToolFunc) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	return t.Fn(ctx, input)
}

// ValidateInput checks required params are present in a JSON object input.
func ValidateInput(def schema.ToolDef, raw json.RawMessage) error {
	if len(def.Required) == 0 {
		return nil
	}
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return schema.NewError(schema.CodeInvalidRequest, fmt.Errorf("tool %q: missing required input: %w", def.Name, schema.ErrInvalidRequest))
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return schema.NewError(schema.CodeInvalidRequest, fmt.Errorf("tool %q: input must be a JSON object: %w", def.Name, schema.ErrInvalidRequest))
	}
	for _, key := range def.Required {
		v, ok := obj[key]
		if !ok || len(v) == 0 || string(v) == "null" || string(v) == `""` {
			return schema.NewError(schema.CodeInvalidRequest, fmt.Errorf("tool %q: missing required param %q: %w", def.Name, key, schema.ErrInvalidRequest))
		}
	}
	return nil
}
