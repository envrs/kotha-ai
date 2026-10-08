package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/llm/tools"
	"github.com/kothagpt/kotha/internal/message"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	"github.com/stretchr/testify/require"
)

type stubTool struct{ info tools.ToolInfo }

func (s stubTool) Info() tools.ToolInfo { return s.info }
func (s stubTool) Run(context.Context, tools.ToolCall) (tools.ToolResponse, error) {
	return tools.ToolResponse{}, nil
}

func textMsg(role message.MessageRole, text string) message.Message {
	return message.Message{Role: role, Parts: []message.ContentPart{message.TextContent{Text: text}}}
}

func TestAnthropicFinishReason(t *testing.T) {
	a := &anthropicClient{}
	cases := map[string]message.FinishReason{
		"end_turn":      message.FinishReasonEndTurn,
		"max_tokens":    message.FinishReasonMaxTokens,
		"tool_use":      message.FinishReasonToolUse,
		"stop_sequence": message.FinishReasonEndTurn,
		"unknown":       message.FinishReasonUnknown,
	}
	for in, want := range cases {
		require.Equal(t, want, a.finishReason(in), in)
	}
}

func TestAnthropicConvertMessages(t *testing.T) {
	a := &anthropicClient{options: anthropicOptions{disableCache: true}}
	msgs := []message.Message{
		textMsg(message.User, "hello"),
		{Role: message.Assistant, Parts: []message.ContentPart{
			message.TextContent{Text: "ok"},
			message.ToolCall{ID: "t1", Name: "read", Input: `{"path":"/x"}`},
		}},
		{Role: message.Tool, Parts: []message.ContentPart{
			message.ToolResult{ToolCallID: "t1", Content: "file contents"},
		}},
	}

	out := a.convertMessages(msgs)
	require.Len(t, out, 3)
	require.Equal(t, anthropic.MessageParamRoleUser, out[0].Role)
	require.Equal(t, anthropic.MessageParamRoleAssistant, out[1].Role)
	require.Equal(t, anthropic.MessageParamRoleUser, out[2].Role) // tool results => user role
}

func TestAnthropicConvertMessagesSkipsBogusAssistant(t *testing.T) {
	a := &anthropicClient{options: anthropicOptions{disableCache: true}}
	// Invalid tool-call JSON and no text => nothing usable => skipped.
	msgs := []message.Message{{Role: message.Assistant, Parts: []message.ContentPart{
		message.ToolCall{ID: "t", Name: "x", Input: `{broken`},
	}}}
	require.Empty(t, a.convertMessages(msgs))
}

func TestAnthropicConvertMessagesImage(t *testing.T) {
	a := &anthropicClient{options: anthropicOptions{disableCache: true}}
	msg := message.Message{Role: message.User, Parts: []message.ContentPart{
		message.TextContent{Text: "what is this"},
		message.BinaryContent{MIMEType: "image/png", Data: []byte{1, 2, 3}},
	}}
	out := a.convertMessages([]message.Message{msg})
	require.Len(t, out, 1)
	require.NotNil(t, out[0].Content[0].OfText)
	require.NotNil(t, out[0].Content[1].OfImage)
}

func TestAnthropicConvertMessagesCachesLastMessages(t *testing.T) {
	a := &anthropicClient{options: anthropicOptions{}} // cache enabled
	msgs := []message.Message{
		textMsg(message.User, "one"),
		textMsg(message.User, "two"),
		textMsg(message.User, "three"),
	}
	out := a.convertMessages(msgs)
	require.Len(t, out, 3)
	require.Empty(t, out[0].Content[0].OfText.CacheControl.Type)
	require.Equal(t, "ephemeral", string(out[2].Content[0].OfText.CacheControl.Type))

	// With cache disabled nothing gets a cache control.
	a2 := &anthropicClient{options: anthropicOptions{disableCache: true}}
	out2 := a2.convertMessages(msgs)
	require.Empty(t, out2[2].Content[0].OfText.CacheControl.Type)
}

func TestAnthropicConvertTools(t *testing.T) {
	a := &anthropicClient{options: anthropicOptions{}} // cache enabled
	base := []tools.BaseTool{
		stubTool{tools.ToolInfo{Name: "one", Description: "first", Parameters: map[string]any{"a": "b"}}},
		stubTool{tools.ToolInfo{Name: "two", Description: "second"}},
	}
	out := a.convertTools(base)
	require.Len(t, out, 2)
	require.Equal(t, "one", out[0].OfTool.Name)
	require.Equal(t, "first", out[0].OfTool.Description.Value)
	require.Equal(t, "ephemeral", string(out[1].OfTool.CacheControl.Type))
	require.Empty(t, out[0].OfTool.CacheControl.Type)

	// Disabled cache means no tool is cached.
	a2 := &anthropicClient{options: anthropicOptions{disableCache: true}}
	out2 := a2.convertTools(base)
	require.Empty(t, out2[1].OfTool.CacheControl.Type)
}

func TestAnthropicToolCalls(t *testing.T) {
	a := &anthropicClient{}
	var msg anthropic.Message
	require.NoError(t, json.Unmarshal([]byte(`{
		"content":[
			{"type":"text","text":"hi"},
			{"type":"tool_use","id":"t1","name":"read","input":{"path":"/x"}}
		]
	}`), &msg))
	calls := a.toolCalls(msg)
	require.Len(t, calls, 1)
	require.Equal(t, "t1", calls[0].ID)
	require.Equal(t, "read", calls[0].Name)
	require.Contains(t, calls[0].Input, "/x")
	require.True(t, calls[0].Finished)
}

func TestAnthropicUsage(t *testing.T) {
	a := &anthropicClient{}
	var msg anthropic.Message
	require.NoError(t, json.Unmarshal([]byte(`{
		"usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":3,"cache_read_input_tokens":4}
	}`), &msg))
	u := a.usage(msg)
	require.Equal(t, TokenUsage{InputTokens: 10, OutputTokens: 20, CacheCreationTokens: 3, CacheReadTokens: 4}, u)
}

func TestAnthropicPreparedMessagesThinking(t *testing.T) {
	thinkOn := anthropicClient{
		providerOptions: providerClientOptions{maxTokens: 1000, model: models.Model{APIModel: "claude-4"}},
		options:         anthropicOptions{shouldThink: DefaultShouldThinkFn},
	}
	user := textMsg(message.User, "please think about this")
	base := anthropic.NewUserMessage(anthropic.NewTextBlock(user.Content().String()))
	out := thinkOn.preparedMessages([]anthropic.MessageParam{base}, nil)
	require.NotNil(t, out.Thinking.OfEnabled)
	require.Equal(t, int64(800), out.Thinking.OfEnabled.BudgetTokens)
	require.Equal(t, float64(1), out.Temperature.Value)

	thinkOff := anthropicClient{
		providerOptions: providerClientOptions{maxTokens: 1000, model: models.Model{APIModel: "claude-4"}},
		options:         anthropicOptions{shouldThink: DefaultShouldThinkFn},
	}
	out2 := thinkOff.preparedMessages([]anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("no special signal here")),
	}, nil)
	require.Nil(t, out2.Thinking.OfEnabled)
	require.Equal(t, float64(0), out2.Temperature.Value)

	noFn := anthropicClient{providerOptions: providerClientOptions{maxTokens: 1000, model: models.Model{APIModel: "x"}}}
	out3 := noFn.preparedMessages([]anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("please think")),
	}, nil)
	require.Nil(t, out3.Thinking.OfEnabled)
}

func TestDefaultShouldThinkFn(t *testing.T) {
	require.True(t, DefaultShouldThinkFn("please think step by step"))
	require.True(t, DefaultShouldThinkFn("THINK"))
	require.False(t, DefaultShouldThinkFn("what is the weather?"))
}

func TestAnthropicOptions(t *testing.T) {
	var opts anthropicOptions
	WithAnthropicBedrock(true)(&opts)
	require.True(t, opts.useBedrock)
	WithAnthropicDisableCache()(&opts)
	require.True(t, opts.disableCache)
	fn := func(string) bool { return true }
	WithAnthropicShouldThinkFn(fn)(&opts)
	require.NotNil(t, opts.shouldThink)
	require.True(t, opts.shouldThink("anything"))
}

func TestOpenAIFinishReason(t *testing.T) {
	o := &openaiClient{}
	cases := map[string]message.FinishReason{
		"stop":       message.FinishReasonEndTurn,
		"length":     message.FinishReasonMaxTokens,
		"tool_calls": message.FinishReasonToolUse,
		"something":  message.FinishReasonUnknown,
	}
	for in, want := range cases {
		require.Equal(t, want, o.finishReason(in), in)
	}
}

func TestOpenAIConvertMessages(t *testing.T) {
	o := &openaiClient{providerOptions: providerClientOptions{systemMessage: "sys"}}
	msgs := []message.Message{
		message.Message{Role: message.User, Parts: []message.ContentPart{
			message.TextContent{Text: "hello"},
			message.BinaryContent{MIMEType: "image/png", Data: []byte{9}},
		}},
		{Role: message.Assistant, Parts: []message.ContentPart{
			message.TextContent{Text: "calling"},
			message.ToolCall{ID: "t1", Name: "read", Input: `{"a":1}`},
		}},
		{Role: message.Tool, Parts: []message.ContentPart{
			message.ToolResult{ToolCallID: "t1", Content: "res1"},
			message.ToolResult{ToolCallID: "t2", Content: "res2"},
		}},
	}
	out := o.convertMessages(msgs)
	require.Len(t, out, 5) // system + user + assistant + 2 tool results
	require.Equal(t, "sys", out[0].OfSystem.Content.OfString.Value)
	userParts := out[1].OfUser.Content.OfArrayOfContentParts
	require.NotNil(t, userParts[0].OfText)
	require.NotNil(t, userParts[1].OfImageURL)
	require.Equal(t, "t1", out[3].OfTool.ToolCallID)
	require.Equal(t, "res2", out[4].OfTool.Content.OfString.Value)
}

func TestOpenAIConvertTools(t *testing.T) {
	o := &openaiClient{}
	out := o.convertTools([]tools.BaseTool{
		stubTool{tools.ToolInfo{Name: "find", Description: "search", Parameters: map[string]any{"q": "x"}, Required: []string{"q"}}},
	})
	require.Len(t, out, 1)
	require.Equal(t, "find", out[0].Function.Name)
	require.Equal(t, "search", out[0].Function.Description.Value)
	require.Equal(t, "object", out[0].Function.Parameters["type"])
	require.Equal(t, []string{"q"}, out[0].Function.Parameters["required"])
}

func TestOpenAIPreparedParams(t *testing.T) {
	t.Run("reasoning model", func(t *testing.T) {
		o := &openaiClient{
			providerOptions: providerClientOptions{
				model:     models.Model{APIModel: "gpt-5", CanReason: true},
				maxTokens: 1000,
			},
			options: openaiOptions{reasoningEffort: "high"},
		}
		params := o.preparedParams(nil, nil)
		require.Equal(t, int64(1000), params.MaxCompletionTokens.Value)
		require.Equal(t, shared.ReasoningEffortHigh, params.ReasoningEffort)
		require.False(t, params.MaxTokens.IsPresent())
	})

	t.Run("reasoning model defaults medium", func(t *testing.T) {
		o := &openaiClient{
			providerOptions: providerClientOptions{model: models.Model{CanReason: true}},
			options:         openaiOptions{reasoningEffort: "bogus"},
		}
		require.Equal(t, shared.ReasoningEffortMedium, o.preparedParams(nil, nil).ReasoningEffort)
	})

	t.Run("non-reasoning model", func(t *testing.T) {
		o := &openaiClient{
			providerOptions: providerClientOptions{model: models.Model{APIModel: "gpt-4.1"}, maxTokens: 500},
		}
		params := o.preparedParams(nil, nil)
		require.Equal(t, int64(500), params.MaxTokens.Value)
		require.False(t, params.MaxCompletionTokens.IsPresent())
	})
}

func TestOpenAIToolCalls(t *testing.T) {
	o := &openaiClient{}
	completion := openai.ChatCompletion{
		Choices: []openai.ChatCompletionChoice{{
			Message: openai.ChatCompletionMessage{
				ToolCalls: []openai.ChatCompletionMessageToolCall{{
					ID: "c1",
					Function: openai.ChatCompletionMessageToolCallFunction{
						Name:      "read",
						Arguments: `{"path":"/x"}`,
					},
				}},
			},
		}},
	}
	calls := o.toolCalls(completion)
	require.Len(t, calls, 1)
	require.Equal(t, "c1", calls[0].ID)
	require.Equal(t, "read", calls[0].Name)
	require.Equal(t, `{"path":"/x"}`, calls[0].Input)

	require.Empty(t, o.toolCalls(openai.ChatCompletion{Choices: []openai.ChatCompletionChoice{{}}}))
}

func TestOpenAIUsage(t *testing.T) {
	u := (&openaiClient{}).usage(openai.ChatCompletion{Usage: openai.CompletionUsage{
		PromptTokens:     100,
		CompletionTokens: 20,
		PromptTokensDetails: openai.CompletionUsagePromptTokensDetails{
			CachedTokens: 30,
		},
	}})
	require.Equal(t, TokenUsage{InputTokens: 70, OutputTokens: 20, CacheReadTokens: 30}, u)
}

func TestOpenAIOptions(t *testing.T) {
	var opts openaiOptions
	WithOpenAIBaseURL("http://x")(&opts)
	require.Equal(t, "http://x", opts.baseURL)
	WithOpenAIDisableCache()(&opts)
	require.True(t, opts.disableCache)
	WithOpenAIExtraHeaders(map[string]string{"X-Title": "Kotha"})(&opts)
	require.Equal(t, "Kotha", opts.extraHeaders["X-Title"])
}

func TestCopilotIsAnthropicModel(t *testing.T) {
	cases := []struct {
		name  string
		model models.ModelID
		want  bool
	}{
		{"claude", models.CopilotClaude4, true},
		{"copilot gpt", models.CopilotGPT4o, false},
	}
	for _, tc := range cases {
		c := &copilotClient{providerOptions: providerClientOptions{model: models.Model{ID: tc.model}}}
		require.Equal(t, tc.want, c.isAnthropicModel(), tc.name)
	}
}
