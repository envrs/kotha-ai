package plugin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHookRegistry_GlobalAndSession(t *testing.T) {
	registry := NewHookRegistry()
	var globalCalled, sessionCalled bool

	registry.RegisterGlobal(func(ctx context.Context, req *PromptRequest) error {
		globalCalled = true
		return nil
	})
	registry.RegisterSession("sess-1", func(ctx context.Context, req *PromptRequest) error {
		sessionCalled = true
		return nil
	})

	hooks := registry.GetHooks("sess-1")
	require.Len(t, hooks, 2)

	hooks = registry.GetHooks("sess-2")
	require.Len(t, hooks, 1)

	// Execute hooks to set flags
	for _, h := range registry.GetHooks("sess-1") {
		require.NoError(t, h(context.Background(), &PromptRequest{}))
	}
	require.True(t, globalCalled)
	require.True(t, sessionCalled)

	registry.ClearSession("sess-1")
	hooks = registry.GetHooks("sess-1")
	require.Len(t, hooks, 1)
}

func TestHookRegistry_SessionHookContext(t *testing.T) {
	registry := NewHookRegistry()
	ctx := WithSessionHooks(context.Background(), registry)
	
	retrieved := GetHookRegistry(ctx)
	require.Equal(t, registry, retrieved)
	
	// No registry in context
	emptyCtx := context.Background()
	require.Nil(t, GetHookRegistry(emptyCtx))
}

func TestAdmissionStore_MarkAndCheck(t *testing.T) {
	store := NewAdmissionStore(1 * time.Hour)
	
	require.False(t, store.IsAdmitted("id-1"))
	store.MarkAdmitted("id-1")
	require.True(t, store.IsAdmitted("id-1"))
	require.False(t, store.IsAdmitted("id-2"))
}

func TestPreparationPipeline_Basic(t *testing.T) {
	registry := NewHookRegistry()
	store := NewAdmissionStore(1 * time.Hour)
	pipeline := NewPreparationPipeline(registry, store)
	
	req := &PromptRequest{
		SessionID:   "sess-1",
		Text:        "hello",
		AdmissionID: "adm-1",
	}
	
	err := pipeline.Prepare(context.Background(), req)
	require.NoError(t, err)
	require.True(t, req.Prepared)
	require.False(t, req.AlreadyAdmitted)
}

func TestPreparationPipeline_Idempotency(t *testing.T) {
	registry := NewHookRegistry()
	store := NewAdmissionStore(1 * time.Hour)
	pipeline := NewPreparationPipeline(registry, store)
	
	req := &PromptRequest{
		SessionID:   "sess-1",
		Text:        "hello",
		AdmissionID: "adm-1",
	}
	
	err := pipeline.Prepare(context.Background(), req)
	require.NoError(t, err)
	require.True(t, req.Prepared)
	
	// Second call with same ID should skip
	req2 := &PromptRequest{
		SessionID:   "sess-1",
		Text:        "hello",
		AdmissionID: "adm-1",
	}
	err = pipeline.Prepare(context.Background(), req2)
	require.NoError(t, err)
	require.True(t, req2.AlreadyAdmitted)
	require.False(t, req2.Prepared) // Not re-prepared
}

func TestPreparationPipeline_Interruptible(t *testing.T) {
	registry := NewHookRegistry()
	store := NewAdmissionStore(1 * time.Hour)
	pipeline := NewPreparationPipeline(registry, store)
	
	// Add a slow stage
	pipeline.AddStage("slow", StageFiles, func(ctx *PrepareContext, stage PrepareStage) error {
		select {
		case <-ctx.Ctx.Done():
			return ctx.Ctx.Err()
		case <-time.After(500 * time.Millisecond):
			return nil
		}
	}, nil)
	
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	req := &PromptRequest{
		SessionID:   "sess-1",
		Text:        "hello",
		AdmissionID: "adm-1",
	}
	
	err := pipeline.Prepare(ctx, req)
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestPreparationPipeline_RevertOnFailure(t *testing.T) {
	registry := NewHookRegistry()
	store := NewAdmissionStore(1 * time.Hour)
	pipeline := NewPreparationPipeline(registry, store)
	
	var stagesRun, stagesReverted []string
	
	pipeline.AddStage("stage1", StageText, func(ctx *PrepareContext, stage PrepareStage) error {
		stagesRun = append(stagesRun, "stage1")
		return nil
	}, func(ctx *PrepareContext, stage PrepareStage) error {
		stagesReverted = append(stagesReverted, "stage1")
		return nil
	})
	
	pipeline.AddStage("stage2", StageFiles, func(ctx *PrepareContext, stage PrepareStage) error {
		stagesRun = append(stagesRun, "stage2")
		return fmt.Errorf("stage2 failed")
	}, func(ctx *PrepareContext, stage PrepareStage) error {
		stagesReverted = append(stagesReverted, "stage2")
		return nil
	})
	
	req := &PromptRequest{
		SessionID:   "sess-1",
		Text:        "hello",
		AdmissionID: "adm-1",
	}
	
	err := pipeline.Prepare(context.Background(), req)
	require.Error(t, err)
	require.Equal(t, []string{"stage1", "stage2"}, stagesRun)
	require.Equal(t, []string{"stage1"}, stagesReverted) // Only stage1 reverted (reverse order)
}

func TestSessionHook(t *testing.T) {
	registry := NewHookRegistry()
	hook := NewSessionHook("sess-1", registry)
	
	var called bool
	hook.Hook(SessionHookKey, func(ctx context.Context, req *PromptRequest) error {
		called = true
		return nil
	})
	
	hooks := registry.GetHooks("sess-1")
	require.Len(t, hooks, 1)
	
	// Actually execute the hook
	require.NoError(t, hooks[0](context.Background(), &PromptRequest{}))
	require.True(t, called)
}

func TestFileDraft(t *testing.T) {
	draft := FileDraft{
		ID:       "fd-1",
		Path:     "/tmp/test.txt",
		Name:     "test.txt",
		Size:     100,
		MIMEType: "text/plain",
		Content:  []byte("hello"),
		Prepared: true,
	}
	require.Equal(t, "fd-1", draft.ID)
	require.True(t, draft.Prepared)
}

func TestAgentMention(t *testing.T) {
	mention := AgentMention{
		Name:      "coder",
		Metadata:  map[string]any{"role": "reviewer"},
		Raw:       "@coder",
		StartPos:  0,
		EndPos:    6,
	}
	require.Equal(t, "coder", mention.Name)
	require.Equal(t, "reviewer", mention.Metadata["role"])
}

func TestSkillRef(t *testing.T) {
	skill := SkillRef{
		Name:     "refactor",
		Args:     map[string]any{"style": "clean"},
		Raw:      "!refactor --style=clean",
		StartPos: 0,
		EndPos:   22,
	}
	require.Equal(t, "refactor", skill.Name)
	require.Equal(t, "clean", skill.Args["style"])
}

var _ = fmt.Errorf