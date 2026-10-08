package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/message"
	"github.com/stretchr/testify/require"
)

// isolateProviderEnv prevents Copilot token discovery from touching the
// real machine (which would trigger a network token exchange).
func isolateProviderEnv(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	for _, k := range []string{"GITHUB_TOKEN", "AWS_REGION", "AWS_DEFAULT_REGION", "AZURE_OPENAI_ENDPOINT", "AZURE_OPENAI_API_VERSION", "AZURE_OPENAI_API_KEY", "VERTEXAI_PROJECT", "VERTEXAI_LOCATION", "LOCAL_ENDPOINT"} {
		t.Setenv(k, "")
	}
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
}

func TestNewProviderSupported(t *testing.T) {
	isolateProviderEnv(t)
	model := models.Model{ID: models.Claude4Sonnet, APIModel: "claude"}
	providers := []models.ModelProvider{
		models.ProviderCopilot,
		models.ProviderAnthropic,
		models.ProviderOpenAI,
		models.ProviderGemini,
		models.ProviderBedrock,
		models.ProviderGROQ,
		models.ProviderAzure,
		models.ProviderVertexAI,
		models.ProviderOpenRouter,
		models.ProviderXAI,
		models.ProviderLocal,
		models.ProviderMock,
	}
	for _, name := range providers {
		t.Run(string(name), func(t *testing.T) {
			p, err := NewProvider(name, WithModel(model))
			require.NoError(t, err)
			require.NotNil(t, p)
			require.Equal(t, model, p.Model())
		})
	}
}

func TestNewProviderUnsupported(t *testing.T) {
	_, err := NewProvider("bogus-provider")
	require.Error(t, err)
	require.Contains(t, err.Error(), "provider not supported")
}

func TestBaseProviderCleansEmptyMessages(t *testing.T) {
	isolateProviderEnv(t)
	p, err := NewProvider(models.ProviderMock, WithModel(models.Model{ID: "m"}))
	require.NoError(t, err)

	bp, ok := p.(*baseProvider[MockClient])
	require.True(t, ok)
	mc := bp.client.(*mockClient)

	msgs := []message.Message{
		{ID: "empty", Parts: nil},
		{ID: "keep", Parts: []message.ContentPart{message.TextContent{Text: "hi"}}},
	}
	_, err = p.SendMessages(context.Background(), msgs, nil)
	require.NoError(t, err)

	require.Len(t, mc.Calls, 1)
	require.Len(t, mc.Calls[0].Messages, 1)
	require.Equal(t, "keep", mc.Calls[0].Messages[0].ID)
}

func TestSendMessagesPropagatesMockError(t *testing.T) {
	isolateProviderEnv(t)
	sentinel := errors.New("boom")
	p, err := NewProvider(models.ProviderMock,
		WithModel(models.Model{ID: "m"}),
		WithMockOptions(WithMockError(sentinel)),
	)
	require.NoError(t, err)

	_, err = p.SendMessages(context.Background(),
		[]message.Message{{ID: "x", Parts: []message.ContentPart{message.TextContent{Text: "hi"}}}}, nil)
	require.ErrorIs(t, err, sentinel)
}

func TestStreamResponseFiltersAndEmitsEvents(t *testing.T) {
	isolateProviderEnv(t)
	p, err := NewProvider(models.ProviderMock,
		WithModel(models.Model{ID: "m"}),
		WithMockOptions(WithMockResponses(&ProviderResponse{
			Content:      "hello",
			ToolCalls:    []message.ToolCall{{ID: "t1", Name: "read"}},
			FinishReason: message.FinishReasonEndTurn,
		})),
	)
	require.NoError(t, err)

	bp := p.(*baseProvider[MockClient])
	mc := bp.client.(*mockClient)

	msgs := []message.Message{
		{ID: "empty", Parts: nil},
		{ID: "keep", Parts: []message.ContentPart{message.TextContent{Text: "hi"}}},
	}
	ch := p.StreamResponse(context.Background(), msgs, nil)

	var types []EventType
	var complete *ProviderResponse
	for ev := range ch {
		types = append(types, ev.Type)
		if ev.Type == EventComplete {
			complete = ev.Response
		}
	}
	require.Contains(t, types, EventContentDelta)
	require.Contains(t, types, EventToolUseStart)
	require.Contains(t, types, EventComplete)
	require.NotNil(t, complete)
	require.Equal(t, "hello", complete.Content)
	require.Len(t, mc.Calls[0].Messages, 1)
}

func TestStreamResponseMockStreamError(t *testing.T) {
	isolateProviderEnv(t)
	sentinel := errors.New("stream blew up")
	p, err := NewProvider(models.ProviderMock,
		WithModel(models.Model{ID: "m"}),
		WithMockOptions(WithMockStreamError(sentinel)),
	)
	require.NoError(t, err)

	ch := p.StreamResponse(context.Background(),
		[]message.Message{{ID: "x", Parts: []message.ContentPart{message.TextContent{Text: "hi"}}}}, nil)
	var got error
	for ev := range ch {
		if ev.Type == EventError {
			got = ev.Error
		}
	}
	require.ErrorIs(t, got, sentinel)
}

func TestMockDefaultResponse(t *testing.T) {
	m := newMockClient(providerClientOptions{}).(*mockClient)
	resp, err := m.next()
	require.NoError(t, err)
	require.Equal(t, message.FinishReasonEndTurn, resp.FinishReason)

	m.Enqueue(&ProviderResponse{Content: "queued"})
	resp, err = m.next()
	require.NoError(t, err)
	require.Equal(t, "queued", resp.Content)
}

func TestProviderClientOptions(t *testing.T) {
	var opts providerClientOptions
	WithAPIKey("k")(&opts)
	WithModel(models.Model{ID: "m"})(&opts)
	WithMaxTokens(42)(&opts)
	WithSystemMessage("sys")(&opts)
	require.Equal(t, "k", opts.apiKey)
	require.Equal(t, models.ModelID("m"), opts.model.ID)
	require.Equal(t, int64(42), opts.maxTokens)
	require.Equal(t, "sys", opts.systemMessage)

	noop := func(*anthropicOptions) {}
	WithAnthropicOptions(noop)(&opts)
	require.Len(t, opts.anthropicOptions, 1)

	WithOpenAIOptions(func(*openaiOptions) {})(&opts)
	require.Len(t, opts.openaiOptions, 1)

	WithGeminiOptions(func(*geminiOptions) {})(&opts)
	require.Len(t, opts.geminiOptions, 1)

	WithBedrockOptions(func(*bedrockOptions) {})(&opts)
	require.Len(t, opts.bedrockOptions, 1)

	WithCopilotOptions(func(*copilotOptions) {})(&opts)
	require.Len(t, opts.copilotOptions, 1)

	WithMockOptions(func(*mockClient) {})(&opts)
	require.Len(t, opts.mockOptions, 1)
}

func TestLocalProviderUsesEndpoint(t *testing.T) {
	isolateProviderEnv(t)
	t.Setenv("LOCAL_ENDPOINT", "http://127.0.0.1:11434/v1")
	p, err := NewProvider(models.ProviderLocal, WithModel(models.Model{ID: "m"}))
	require.NoError(t, err)
	require.NotNil(t, p)
}
