package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupportedModelsPopulated(t *testing.T) {
	require.NotEmpty(t, SupportedModels)
	require.Contains(t, SupportedModels, BedrockClaude37Sonnet)
	for id, m := range SupportedModels {
		require.Equal(t, id, m.ID)
		require.NotEmpty(t, string(m.Provider), "model %s missing provider", id)
		require.NotEmpty(t, m.APIModel, "model %s missing api_model", id)
		require.NotEmpty(t, m.Name, "model %s missing name", id)
	}
}

func TestProviderPopularityCoversKnown(t *testing.T) {
	for _, p := range []ModelProvider{ProviderAnthropic, ProviderOpenAI, ProviderGemini, ProviderGROQ, ProviderOpenRouter, ProviderAzure, ProviderVertexAI, ProviderCopilot, ProviderBedrock} {
		_, ok := ProviderPopularity[p]
		require.True(t, ok, "provider %s missing popularity", p)
	}
}
