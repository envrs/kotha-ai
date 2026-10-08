package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestAllowBurstThenDenies(t *testing.T) {
	l := New(0, 2) // fixed budget, no refill
	for i := 0; i < 2; i++ {
		if !l.Allow() {
			t.Fatalf("expected burst token %d", i+1)
		}
	}
	if l.Allow() {
		t.Fatal("expected denial after budget spent")
	}
}

func TestRefill(t *testing.T) {
	l := New(1000, 1)
	if !l.Allow() {
		t.Fatal("expected first allow")
	}
	if l.Allow() {
		t.Fatal("expected denial before refill")
	}
	time.Sleep(5 * time.Millisecond)
	if !l.Allow() {
		t.Fatal("expected refill")
	}
}

func TestWaitRespectsCancel(t *testing.T) {
	l := New(0, 1)
	_ = l.Allow() // drain
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := l.Wait(ctx); err == nil {
		t.Fatal("expected ctx error")
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry(0, 1)
	if !r.Allow("a") || r.Allow("a") {
		t.Fatal("per-key budget broken")
	}
	if !r.Allow("b") {
		t.Fatal("keys must be independent")
	}
}
