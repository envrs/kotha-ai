package testllm

import (
	"context"
	"testing"

	"github.com/kothagpt/kotha/internal/llm/provider"
	"github.com/kothagpt/kotha/internal/llm/tools"
	"github.com/kothagpt/kotha/internal/message"
	"github.com/stretchr/testify/require"
)

// TestController_IntegrationWithSDK demonstrates using TestLLMController with an SDK-like client.
func TestController_IntegrationWithSDK(t *testing.T) {
	ctrl := NewTestLLMController()

	ctrl.AddResponse(&provider.ProviderResponse{
		Content:      "Hello from test LLM!",
		FinishReason: message.FinishReasonEndTurn,
	})

	// Build provider using the controller
	testProv := ctrl.BuildProvider("test-model")

	// Verify the provider works directly (SDK would use this internally)
	resp, err := testProv.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "hi"}}}},
		nil)
	require.NoError(t, err)
	require.Equal(t, "Hello from test LLM!", resp.Content)
}

// TestController_RequestDependentBehavior demonstrates request-dependent responses.
func TestController_RequestDependentBehavior(t *testing.T) {
	ctrl := NewTestLLMController()

	// Different responses for different queries
	ctrl.AddScript(
		MatchContent("capital of France"),
		&provider.ProviderResponse{Content: "Paris is the capital of France."},
		nil,
	)
	ctrl.AddScript(
		MatchContent("capital of Japan"),
		&provider.ProviderResponse{Content: "Tokyo is the capital of Japan."},
		nil,
	)
	ctrl.SetFallback(&provider.ProviderResponse{Content: "I don't know that capital."}, nil)

	prov := ctrl.BuildProvider("test-model")

	// Test France
	resp1, err := prov.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "What is the capital of France?"}}}}, nil)
	require.NoError(t, err)
	require.Contains(t, resp1.Content, "Paris")

	// Test Japan
	resp2, err := prov.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "What is the capital of Japan?"}}}}, nil)
	require.NoError(t, err)
	require.Contains(t, resp2.Content, "Tokyo")

	// Test unknown
	resp3, err := prov.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "What is the capital of Mars?"}}}}, nil)
	require.NoError(t, err)
	require.Contains(t, resp3.Content, "don't know")
}

// TestController_CustomMatchers demonstrates custom request matchers.
func TestController_CustomMatchers(t *testing.T) {
	ctrl := NewTestLLMController()

	// Custom matcher for specific tool - matches any tool call
	ctrl.AddScript(
		func(_ []message.Message, tools []tools.BaseTool) bool {
			return len(tools) > 0
		},
		&provider.ProviderResponse{Content: "Custom tool response"},
		nil,
	)

	prov := ctrl.BuildProvider("test-model")

	// Use a tool that will match
	resp, err := prov.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "use custom tool"}}}},
		[]tools.BaseTool{tools.NewBashTool(nil)},
	)
	require.NoError(t, err)
	require.Equal(t, "Custom tool response", resp.Content)
}