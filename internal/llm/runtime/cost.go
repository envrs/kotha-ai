package runtime

import (
	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/llm/schema"
)

const tokensPerMillion = 1_000_000

// CostFor computes the USD cost of a usage record against the model's
// pricing table. Unknown models cost 0 (pricing is optional metadata).
//
// Cache pricing conventions across provider tables differ:
//   - CostPer1MInCached: cache-write price for Anthropic-style tables,
//     cache-read price for OpenAI-style tables.
//   - CostPer1MOutCached: cache-read price when non-zero (Anthropic).
//
// So cache reads are billed at CostPer1MOutCached when set, otherwise at
// CostPer1MInCached, otherwise at the base input rate.
func CostFor(modelID string, u schema.TokenUsage) float64 {
	m, ok := models.SupportedModels[models.ModelID(modelID)]
	if !ok {
		return 0
	}

	input, read, created := u.Input, u.CacheRead, u.CacheCreated
	switch m.Provider {
	case models.ProviderAnthropic, models.ProviderBedrock:
		// Anthropic-style APIs report input_tokens excluding cache tokens;
		// bill cache tokens separately on top of input.
	default:
		// OpenAI-style APIs report cached tokens as a subset of
		// prompt_tokens; subtract them so they are not billed twice.
		if cache := read + created; cache > 0 && cache <= input {
			input -= cache
		}
	}

	readRate := m.CostPer1MOutCached
	if readRate <= 0 {
		readRate = m.CostPer1MInCached
	}
	if readRate <= 0 {
		readRate = m.CostPer1MIn
	}
	createdRate := m.CostPer1MInCached
	if createdRate <= 0 {
		createdRate = m.CostPer1MIn
	}

	cost := float64(input)*m.CostPer1MIn +
		float64(read)*readRate +
		float64(created)*createdRate +
		float64(u.Output)*m.CostPer1MOut
	return cost / tokensPerMillion
}

// CostForCompletion computes the cost of a completed model turn.
func CostForCompletion(modelID string, c schema.Completion) float64 {
	return CostFor(modelID, c.Usage)
}
