package message

import (
	"testing"

	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/stretchr/testify/require"
)

func TestContentAccessors(t *testing.T) {
	m := &Message{Parts: []ContentPart{
		TextContent{Text: "hello"},
		ReasoningContent{Thinking: "hmm"},
		ImageURLContent{URL: "http://x/y.png"},
		BinaryContent{MIMEType: "image/png", Data: []byte("abc")},
		ToolCall{ID: "c1", Name: "bash"},
		ToolResult{ToolCallID: "c1", Content: "ok"},
		Finish{Reason: FinishReasonEndTurn},
	}}
	require.Equal(t, "hello", m.Content().Text)
	require.Equal(t, "hmm", m.ReasoningContent().Thinking)
	require.Len(t, m.ImageURLContent(), 1)
	require.Len(t, m.BinaryContent(), 1)
	require.Len(t, m.ToolCalls(), 1)
	require.Len(t, m.ToolResults(), 1)
	require.True(t, m.IsFinished())
	require.Equal(t, FinishReasonEndTurn, m.FinishReason())
	require.NotNil(t, m.FinishPart())
	require.False(t, m.IsThinking())

	empty := &Message{}
	require.Equal(t, "", empty.Content().Text)
	require.Nil(t, empty.FinishPart())
	require.False(t, empty.IsFinished())
}

func TestAppendAndMutations(t *testing.T) {
	m := &Message{}
	m.AppendContent("a")
	m.AppendContent("b")
	require.Equal(t, "ab", m.Content().Text)

	m2 := &Message{}
	m2.AppendReasoningContent("x")
	m2.AppendReasoningContent("y")
	require.Equal(t, "xy", m2.ReasoningContent().Thinking)

	m3 := &Message{Parts: []ContentPart{ToolCall{ID: "1", Input: "a"}}}
	m3.AppendToolCallInput("1", "b")
	require.Equal(t, "ab", m3.ToolCalls()[0].Input)
	m3.FinishToolCall("1")
	require.True(t, m3.ToolCalls()[0].Finished)
	m3.FinishToolCall("missing") // no-op

	m3.AddToolCall(ToolCall{ID: "1", Input: "z"})
	require.Equal(t, "z", m3.ToolCalls()[0].Input)
	m3.AddToolCall(ToolCall{ID: "2"})
	require.Len(t, m3.ToolCalls(), 2)
	m3.SetToolCalls([]ToolCall{{ID: "9"}})
	require.Len(t, m3.ToolCalls(), 1)

	m3.AddToolResult(ToolResult{ToolCallID: "9"})
	m3.SetToolResults([]ToolResult{{ToolCallID: "8"}})
	require.Len(t, m3.ToolResults(), 2)

	m4 := &Message{}
	m4.AddFinish(FinishReasonError)
	m4.AddFinish(FinishReasonCanceled) // replaces
	require.Len(t, []ContentPart{m4.FinishPart()}, 1)
	require.Equal(t, FinishReasonCanceled, m4.FinishReason())

	m4.AddImageURL("u", "auto")
	m4.AddBinary("text/plain", []byte("d"))
	require.Len(t, m4.ImageURLContent(), 1)
	require.Len(t, m4.BinaryContent(), 1)
}

func TestMarshallRoundtrip(t *testing.T) {
	parts := []ContentPart{
		TextContent{Text: "hi"},
		ReasoningContent{Thinking: "t"},
		ImageURLContent{URL: "u"},
		BinaryContent{Path: "p", MIMEType: "m", Data: []byte("d")},
		ToolCall{ID: "c", Name: "n", Input: "i"},
		ToolResult{ToolCallID: "c", Content: "o"},
		Finish{Reason: "stop"},
	}
	data, err := marshallParts(parts)
	require.NoError(t, err)
	back, err := unmarshallParts(data)
	require.NoError(t, err)
	require.Len(t, back, len(parts))

	_, err = marshallParts([]ContentPart{badPart{}})
	require.Error(t, err)
	_, err = unmarshallParts([]byte("not json"))
	require.Error(t, err)
	_, err = unmarshallParts([]byte(`[{"type":"nope","data":{}}]`))
	require.Error(t, err)
}

type badPart struct{}

func (badPart) isPart() {}

func TestBinaryString(t *testing.T) {
	b := BinaryContent{MIMEType: "image/png", Data: []byte("hi")}
	require.Contains(t, b.String(models.ProviderOpenAI), "data:image/png;base64,")
	require.NotContains(t, b.String("other"), "data:")
	require.Equal(t, "u", ImageURLContent{URL: "u"}.String())
	require.Equal(t, "t", TextContent{Text: "t"}.String())
	require.Equal(t, "th", ReasoningContent{Thinking: "th"}.String())
}

func TestIsThinking(t *testing.T) {
	m := &Message{Parts: []ContentPart{ReasoningContent{Thinking: "x"}}}
	require.True(t, m.IsThinking())
}
