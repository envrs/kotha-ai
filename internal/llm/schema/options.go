package schema

import "time"

// RequestOptions is the canonical per-request LLM configuration.
// Providers translate this into their native request shape.
type RequestOptions struct {
	Model       string         `json:"model"`
	Provider    string         `json:"provider,omitempty"`
	System      string         `json:"system,omitempty"`
	MaxTokens   int64          `json:"max_tokens,omitempty"`
	Temperature *float64       `json:"temperature,omitempty"`
	TopP        *float64       `json:"top_p,omitempty"`
	Timeout     time.Duration  `json:"timeout,omitempty"`
	MaxRetries  int            `json:"max_retries,omitempty"`
	Tools       []ToolDef      `json:"tools,omitempty"`
	Extra       map[string]any `json:"extra,omitempty"`
}

// ToolDef describes a callable tool in provider-neutral form.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Required    []string       `json:"required,omitempty"`
}

const (
	DefaultMaxTokens  = 8192
	DefaultMaxRetries = 8
	DefaultTimeout    = 5 * time.Minute
)

type Option func(*RequestOptions)

func WithModel(model string) Option {
	return func(o *RequestOptions) { o.Model = model }
}

func WithProvider(provider string) Option {
	return func(o *RequestOptions) { o.Provider = provider }
}

func WithSystem(system string) Option {
	return func(o *RequestOptions) { o.System = system }
}

func WithMaxTokens(n int64) Option {
	return func(o *RequestOptions) { o.MaxTokens = n }
}

func WithTemperature(t float64) Option {
	return func(o *RequestOptions) { o.Temperature = &t }
}

func WithTopP(p float64) Option {
	return func(o *RequestOptions) { o.TopP = &p }
}

func WithTimeout(d time.Duration) Option {
	return func(o *RequestOptions) { o.Timeout = d }
}

func WithMaxRetries(n int) Option {
	return func(o *RequestOptions) { o.MaxRetries = n }
}

func WithTools(tools ...ToolDef) Option {
	return func(o *RequestOptions) { o.Tools = append(o.Tools, tools...) }
}

func WithExtra(key string, value any) Option {
	return func(o *RequestOptions) {
		if o.Extra == nil {
			o.Extra = map[string]any{}
		}
		o.Extra[key] = value
	}
}

// NewRequestOptions applies opts over sane defaults.
func NewRequestOptions(opts ...Option) RequestOptions {
	o := RequestOptions{MaxTokens: DefaultMaxTokens, MaxRetries: DefaultMaxRetries, Timeout: DefaultTimeout}
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

// Validate ensures required fields are present and bounded.
func (o RequestOptions) Validate() error {
	if o.Model == "" {
		return NewError(CodeInvalidRequest, ErrInvalidRequest)
	}
	if o.MaxTokens < 0 {
		return NewError(CodeInvalidRequest, ErrInvalidRequest)
	}
	if o.Temperature != nil && (*o.Temperature < 0 || *o.Temperature > 2) {
		return NewError(CodeInvalidRequest, ErrInvalidRequest)
	}
	if o.MaxRetries < 0 {
		return NewError(CodeInvalidRequest, ErrInvalidRequest)
	}
	for _, t := range o.Tools {
		if t.Name == "" {
			return NewError(CodeInvalidRequest, ErrInvalidRequest)
		}
	}
	return nil
}
