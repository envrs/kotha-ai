package schema

import (
	"errors"
	"testing"
)

func TestIDs(t *testing.T) {
	if err := SessionID("").Validate(); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("expected ErrInvalidID, got %v", err)
	}
	if NewSessionID().Validate() != nil {
		t.Fatal("expected valid session id")
	}
}

func TestMessageValidate(t *testing.T) {
	m := NewUserMessage("hi")
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := Message{Role: "nope"}
	if bad.Validate() == nil {
		t.Fatal("expected invalid role error")
	}
}

func TestOptionsValidate(t *testing.T) {
	o := NewRequestOptions(WithModel("m"))
	if err := o.Validate(); err != nil {
		t.Fatal(err)
	}
	if (RequestOptions{}).Validate() == nil {
		t.Fatal("expected model required error")
	}
}

func TestErrorHelpers(t *testing.T) {
	err := WrapRateLimited("openai", "gpt", 0, ErrRateLimited)
	if !IsRetryable(err) {
		t.Fatal("expected retryable")
	}
	if CodeOf(err) != CodeRateLimited {
		t.Fatalf("unexpected code %s", CodeOf(err))
	}
}
