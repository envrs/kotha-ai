package testllm

import (
	"context"
	"errors"
	"testing"

	"github.com/kothagpt/kotha/internal/llm/provider"
	"github.com/kothagpt/kotha/internal/llm/tools"
	"github.com/kothagpt/kotha/internal/message"
	"github.com/stretchr/testify/require"
)

func TestController_SequentialResponses(t *testing.T) {
	ctrl := NewTestLLMController()

	// Add sequential responses
	ctrl.AddResponse(&provider.ProviderResponse{Content: "first"})
	ctrl.AddResponse(&provider.ProviderResponse{Content: "second"})
	ctrl.AddResponse(&provider.ProviderResponse{Content: "third"})

	prov := ctrl.BuildProvider("test-model")

	// First call gets first response
	resp1, err := prov.SendMessages(context.Background(), []message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "hello"}}}}, nil)
	require.NoError(t, err)
	require.Equal(t, "first", resp1.Content)

	// Second call gets second response
	resp2, err := prov.SendMessages(context.Background(), []message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "hello"}}}}, nil)
	require.NoError(t, err)
	require.Equal(t, "second", resp2.Content)

	// Third call gets third response
	resp3, err := prov.SendMessages(context.Background(), []message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "hello"}}}}, nil)
	require.NoError(t, err)
	require.Equal(t, "third", resp3.Content)

	// Fourth call gets fallback (empty response)
	resp4, err := prov.SendMessages(context.Background(), []message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "hello"}}}}, nil)
	require.NoError(t, err)
	require.Equal(t, "", resp4.Content)
}

func TestController_RequestMatching(t *testing.T) {
	ctrl := NewTestLLMController()

	// Add request-specific responses
	ctrl.AddScript(MatchContent("hello"), &provider.ProviderResponse{Content: "Hello there!"}, nil)
	ctrl.AddScript(MatchContent("goodbye"), &provider.ProviderResponse{Content: "See you later!"}, nil)
	ctrl.SetFallback(&provider.ProviderResponse{Content: "I don't understand"}, nil)

	prov := ctrl.BuildProvider("test-model")

	// Request matching "hello"
	resp1, err := prov.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "hello world"}}}}, nil)
	require.NoError(t, err)
	require.Equal(t, "Hello there!", resp1.Content)

	// Request matching "goodbye"
	resp2, err := prov.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "goodbye friend"}}}}, nil)
	require.NoError(t, err)
	require.Equal(t, "See you later!", resp2.Content)

	// Request matching nothing
	resp3, err := prov.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "random"}}}}, nil)
	require.NoError(t, err)
	require.Equal(t, "I don't understand", resp3.Content)
}

func TestController_Streaming(t *testing.T) {
	ctrl := NewTestLLMController()

	ctrl.AddResponse(&provider.ProviderResponse{
		Content: "Hello",
		FinishReason: message.FinishReasonEndTurn,
	})

	prov := ctrl.BuildProvider("test-model")

	ctx := context.Background()
	ch := prov.StreamResponse(ctx, []message.Message{
		{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "hi"}}},
	}, nil)

	var events []provider.ProviderEvent
	for ev := range ch {
		events = append(events, ev)
	}

	require.Len(t, events, 2) // ContentDelta + Complete
	require.Equal(t, provider.EventContentDelta, events[0].Type)
	require.Equal(t, "Hello", events[0].Content)
	require.Equal(t, provider.EventComplete, events[1].Type)
	require.NotNil(t, events[1].Response)
	require.Equal(t, "Hello", events[1].Response.Content)
}

func TestController_CallRecording(t *testing.T) {
	ctrl := NewTestLLMController()
	ctrl.AddResponse(&provider.ProviderResponse{Content: "response"})

	prov := ctrl.BuildProvider("test-model")

	// Make a call
	_, err := prov.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "test"}}}}, nil)
	require.NoError(t, err)

	// Check recorded calls
	calls := ctrl.Calls()
	require.Len(t, calls, 1)
	require.Equal(t, "test", calls[0].Messages[0].Content().String())
	require.Equal(t, "response", calls[0].Response.Content)
}

func TestController_Assertions(t *testing.T) {
	ctrl := NewTestLLMController()
	ctrl.AddResponse(&provider.ProviderResponse{Content: "test"})

	prov := ctrl.BuildProvider("test-model")
	_, _ = prov.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "test"}}}}, nil)

	// Test assertions
	call := ctrl.ExpectCall()
	require.Equal(t, "test", call.Response.Content)

	// Test ExpectCalls
	ctrl2 := NewTestLLMController()
	ctrl2.AddResponse(&provider.ProviderResponse{Content: "1"})
	ctrl2.AddResponse(&provider.ProviderResponse{Content: "2"})

	prov2 := ctrl2.BuildProvider("test")
	_, _ = prov2.SendMessages(context.Background(), []message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "a"}}}}, nil)
	_, _ = prov2.SendMessages(context.Background(), []message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "b"}}}}, nil)

	calls := ctrl2.ExpectCalls(2)
	require.Len(t, calls, 2)

	// Test ExpectNoCalls
	ctrl3 := NewTestLLMController()
	ctrl3.ExpectNoCalls()

	// Test ExpectCallMatching
	ctrl4 := NewTestLLMController()
	ctrl4.AddResponse(&provider.ProviderResponse{Content: "match"})
	prov4 := ctrl4.BuildProvider("test")
	_, _ = prov4.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "find me"}}}}, nil)

	call4 := ctrl4.ExpectCallMatching(func(call MockCall) bool {
		return call.Messages[0].Content().String() == "find me"
	})
	require.Equal(t, "match", call4.Response.Content)
}

func TestController_ToolMatching(t *testing.T) {
	ctrl := NewTestLLMController()

	// Mock tool
	mockTool := tools.NewBashTool(nil)

	ctrl.AddScript(MatchToolCall("bash"), &provider.ProviderResponse{
		Content: "ran bash",
		ToolCalls: []message.ToolCall{{Name: "bash", Input: `{"command":"ls"}`}},
	}, nil)

	prov := ctrl.BuildProvider("test-model")

	// Call with bash tool
	resp, err := prov.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "run bash"}}}},
		[]tools.BaseTool{mockTool},
	)
	require.NoError(t, err)
	require.Equal(t, "ran bash", resp.Content)
	require.Len(t, resp.ToolCalls, 1)
	require.Equal(t, "bash", resp.ToolCalls[0].Name)
}

func TestController_ConsumeOnMatch(t *testing.T) {
	ctrl := NewTestLLMController().WithConsumeOnMatch(false)

	ctrl.AddScript(MatchContent("test"), &provider.ProviderResponse{Content: "reusable"}, nil)

	prov := ctrl.BuildProvider("test-model")

	// First call
	resp1, _ := prov.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "test"}}}}, nil)
	require.Equal(t, "reusable", resp1.Content)

	// Second call should get same response (not consumed)
	resp2, _ := prov.SendMessages(context.Background(),
		[]message.Message{{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "test"}}}}, nil)
	require.Equal(t, "reusable", resp2.Content)
}

func TestController_StreamError(t *testing.T) {
	ctrl := NewTestLLMController()
	ctrl.SetStreamError(errors.New("stream failed"))

	prov := ctrl.BuildProvider("test-model")

	ctx := context.Background()
	ch := prov.StreamResponse(ctx, []message.Message{
		{Role: message.User, Parts: []message.ContentPart{message.TextContent{Text: "hi"}}},
	}, nil)

	var events []provider.ProviderEvent
	for ev := range ch {
		events = append(events, ev)
	}

	require.Len(t, events, 1)
	require.Equal(t, provider.EventError, events[0].Type)
	require.Error(t, events[0].Error)
	require.Equal(t, "stream failed", events[0].Error.Error())
}

var _ = errors.New