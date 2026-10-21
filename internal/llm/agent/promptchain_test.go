package agent

import (
	"context"
	"testing"

	"github.com/kothagpt/kotha/internal/plugin"
	"github.com/stretchr/testify/require"
)

func TestPromptChainAdmits(t *testing.T) {
	chain := plugin.NewChain()
	var seen string
	chain.Use(plugin.LegacyFunc("record", func(ctx context.Context, req *plugin.PromptRequest) error {
		seen = req.Content
		req.Content = req.Content + " [seen]"
		return nil
	}))

	// Build a minimal agent with the chain installed.
	ag := &agent{promptChain: chain, logger: nil}
	// Need a non-nil logger? admitPrompt doesn't use it.
	// Use a real agent via NewAgent? requires provider. Instead test admitPrompt directly.
	got, _, err := ag.admitPrompt(context.Background(), "s1", "hello", nil)
	require.NoError(t, err)
	require.Equal(t, "hello [seen]", got)
	require.Equal(t, "hello", seen)
}

func TestPromptChainBlocks(t *testing.T) {
	chain := plugin.NewChain()
	chain.Use(plugin.LegacyFunc("blocker", func(ctx context.Context, req *plugin.PromptRequest) error {
		return plugin.Block("blocker", "not allowed")
	}))
	ag := &agent{promptChain: chain}
	_, _, err := ag.admitPrompt(context.Background(), "s1", "hello", nil)
	require.Error(t, err)
	var be *plugin.BlockedError
	require.ErrorAs(t, err, &be)
	require.Equal(t, "blocker", be.Interceptor)
	require.Equal(t, "not allowed", be.Reason)
}

func TestPromptChainNilIsNoop(t *testing.T) {
	ag := &agent{}
	got, _, err := ag.admitPrompt(context.Background(), "s1", "hello", nil)
	require.NoError(t, err)
	require.Equal(t, "hello", got)
}
