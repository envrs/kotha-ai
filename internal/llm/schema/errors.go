package schema

import (
	"errors"
	"fmt"
	"time"
)

// Canonical LLM error codes shared by all providers.
type ErrorCode string

const (
	CodeUnknown          ErrorCode = "unknown"
	CodeInvalidID        ErrorCode = "invalid_id"
	CodeInvalidRequest   ErrorCode = "invalid_request"
	CodeAuth             ErrorCode = "auth"
	CodeNotFound         ErrorCode = "not_found"
	CodeRateLimited      ErrorCode = "rate_limited"
	CodeContextTooLarge  ErrorCode = "context_too_large"
	CodeMaxRetries       ErrorCode = "max_retries"
	CodeCancelled        ErrorCode = "cancelled"
	CodeBusy             ErrorCode = "session_busy"
	CodePermissionDenied ErrorCode = "permission_denied"
	CodeProvider         ErrorCode = "provider"
)

var (
	ErrInvalidID        = errors.New("invalid id")
	ErrInvalidRequest   = errors.New("invalid request")
	ErrAuth             = errors.New("authentication failed")
	ErrNotFound         = errors.New("not found")
	ErrRateLimited      = errors.New("rate limited")
	ErrContextTooLarge  = errors.New("context too large")
	ErrMaxRetries       = errors.New("maximum retry attempts reached")
	ErrRequestCancelled = errors.New("request cancelled by user")
	ErrSessionBusy      = errors.New("session is currently processing another request")
	ErrPermissionDenied = errors.New("permission denied")
)

// LLMError is a structured, wrappable provider/agent error.
type LLMError struct {
	Code       ErrorCode
	Provider   string
	Model      string
	Retryable  bool
	RetryAfter time.Duration
	Err        error
}

func (e *LLMError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s (provider=%s model=%s): %v", e.Code, e.Provider, e.Model, e.Err)
	}
	return fmt.Sprintf("%s (provider=%s model=%s)", e.Code, e.Provider, e.Model)
}

func (e *LLMError) Unwrap() error { return e.Err }

// WithRetryAfter marks the error retryable after d.
func (e *LLMError) WithRetryAfter(d time.Duration) *LLMError {
	e.Retryable = true
	e.RetryAfter = d
	return e
}

func NewError(code ErrorCode, err error) *LLMError {
	return &LLMError{Code: code, Err: err}
}

func WrapProvider(provider, model string, err error) *LLMError {
	return &LLMError{Code: CodeProvider, Provider: provider, Model: model, Err: err}
}

func WrapRateLimited(provider, model string, retryAfter time.Duration, err error) *LLMError {
	return &LLMError{
		Code: CodeRateLimited, Provider: provider, Model: model,
		Retryable: true, RetryAfter: retryAfter, Err: err,
	}
}

// IsRetryable reports whether err is a retryable LLMError.
func IsRetryable(err error) bool {
	var le *LLMError
	if errors.As(err, &le) {
		return le.Retryable
	}
	return errors.Is(err, ErrRateLimited)
}

// CodeOf extracts the ErrorCode, defaulting to CodeUnknown.
func CodeOf(err error) ErrorCode {
	var le *LLMError
	if errors.As(err, &le) {
		return le.Code
	}
	switch {
	case errors.Is(err, ErrInvalidID):
		return CodeInvalidID
	case errors.Is(err, ErrInvalidRequest):
		return CodeInvalidRequest
	case errors.Is(err, ErrAuth):
		return CodeAuth
	case errors.Is(err, ErrNotFound):
		return CodeNotFound
	case errors.Is(err, ErrRateLimited):
		return CodeRateLimited
	case errors.Is(err, ErrContextTooLarge):
		return CodeContextTooLarge
	case errors.Is(err, ErrMaxRetries):
		return CodeMaxRetries
	case errors.Is(err, ErrRequestCancelled):
		return CodeCancelled
	case errors.Is(err, ErrSessionBusy):
		return CodeBusy
	case errors.Is(err, ErrPermissionDenied):
		return CodePermissionDenied
	default:
		return CodeUnknown
	}
}
