package runtime

import (
	"github.com/kothagpt/kotha/internal/llm/schema"
)

// Token estimation uses the common ~4 bytes per token heuristic. It is
// intentionally cheap: budgeting decisions need order-of-magnitude
// accuracy, not tokenizer parity.

const bytesPerToken = 4

// messageOverhead approximates per-message framing tokens (role, delimiters).
const messageOverhead = 4

// EstimateTokens approximates the token count of s.
func EstimateTokens(s string) int64 {
	if s == "" {
		return 0
	}
	return int64((len(s) + bytesPerToken - 1) / bytesPerToken)
}

// EstimateMessageTokens approximates the token cost of one message,
// including text, tool inputs/results, and fixed framing overhead.
func EstimateMessageTokens(m schema.Message) int64 {
	total := int64(messageOverhead)
	for _, b := range m.Blocks {
		switch b.Type {
		case schema.BlockText:
			total += EstimateTokens(b.Text)
		case schema.BlockToolCall:
			total += EstimateTokens(b.Name) + EstimateTokens(b.Input)
		case schema.BlockToolResult:
			total += EstimateTokens(b.Content)
		case schema.BlockReasoning:
			total += EstimateTokens(b.Thinking)
		}
	}
	return total
}

// EstimateMessagesTokens approximates the token cost of a transcript.
func EstimateMessagesTokens(msgs []schema.Message) int64 {
	var total int64
	for _, m := range msgs {
		total += EstimateMessageTokens(m)
	}
	return total
}
