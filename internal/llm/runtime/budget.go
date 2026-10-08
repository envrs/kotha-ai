package runtime

import "github.com/kothagpt/kotha/internal/llm/schema"

// TrimToBudget trims a transcript so its estimated token count fits
// maxTokens. Policy, in order of protection:
//
//  1. System messages are always kept (configuration, not history).
//  2. The final message is always kept (the request being answered).
//  3. Oldest remaining messages are dropped first. Dropping an assistant
//     message that requested tools also drops its tool-result messages so
//     no orphan results remain.
//
// It returns the trimmed transcript and how many messages were dropped.
// The essential messages are always returned, even when they alone
// exceed the budget — callers decide whether to fail or proceed over.
func TrimToBudget(msgs []schema.Message, maxTokens int64) ([]schema.Message, int) {
	if maxTokens <= 0 {
		return msgs, 0
	}
	total := EstimateMessagesTokens(msgs)
	if total <= maxTokens {
		return msgs, 0
	}

	n := len(msgs)
	if n == 0 {
		return msgs, 0
	}
	essential := make([]bool, n)
	for i, m := range msgs {
		if m.Role == schema.RoleSystem {
			essential[i] = true
		}
	}
	essential[n-1] = true

	keep := make([]bool, n)
	for i := range keep {
		keep[i] = true
	}
	budgeted := total

	dropped := 0
	for i := 0; i < n && budgeted > maxTokens; i++ {
		if essential[i] || !keep[i] {
			continue
		}
		keep[i] = false
		budgeted -= EstimateMessageTokens(msgs[i])
		dropped++
		// Drop tool results tied to an assistant tool-call message.
		if msgs[i].Role == schema.RoleAssistant && len(msgs[i].ToolCalls()) > 0 {
			for j := i + 1; j < n && msgs[j].Role == schema.RoleTool && !essential[j]; j++ {
				if keep[j] {
					keep[j] = false
					budgeted -= EstimateMessageTokens(msgs[j])
					dropped++
				}
			}
		}
	}

	if dropped == 0 {
		return msgs, 0
	}
	out := make([]schema.Message, 0, n-dropped)
	for i, m := range msgs {
		if keep[i] {
			out = append(out, m)
		}
	}
	return out, dropped
}
