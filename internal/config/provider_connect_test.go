package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/stretchr/testify/require"
)

// setCfgWithHome installs a config with a writable HOME dir and restores the
// previous global state. It mirrors the helpers used by other config tests.
func setCfgWithHome(t *testing.T, c *Config) *Config {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("LOCALAPPDATA", dir)
	old := cfg
	cfg = c
	t.Cleanup(func() { cfg = old })
	oldProv := defaultProviderInstance
	SetDefaultProvider(NewDefaultProvider(c))
	t.Cleanup(func() { SetDefaultProvider(oldProv) })
	return c
}

func TestSetProviderEnableWithKey(t *testing.T) {
	isolateEnv(t)
	base := &Config{
		WorkingDir: t.TempDir(),
		Providers:  map[models.ModelProvider]Provider{},
		Agents:     map[AgentName]Agent{},
	}
	setCfgWithHome(t, base)

	p := GetDefaultProvider()
	require.NoError(t, p.SetProvider(models.ProviderAnthropic, true, "sk-test"))

	require.False(t, base.Providers[models.ProviderAnthropic].Disabled)
	require.Equal(t, "sk-test", base.Providers[models.ProviderAnthropic].APIKey)

	data, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".kotha.json"))
	require.NoError(t, err)
	var onDisk Config
	require.NoError(t, json.Unmarshal(data, &onDisk))
	require.Equal(t, "sk-test", onDisk.Providers[models.ProviderAnthropic].APIKey)
}

func TestSetProviderDisableClearsKey(t *testing.T) {
	isolateEnv(t)
	base := &Config{
		WorkingDir: t.TempDir(),
		Providers: map[models.ModelProvider]Provider{
			models.ProviderAnthropic: {APIKey: "sk-test"},
		},
		Agents: map[AgentName]Agent{},
	}
	setCfgWithHome(t, base)

	p := GetDefaultProvider()
	require.NoError(t, p.SetProvider(models.ProviderAnthropic, false, ""))

	require.True(t, base.Providers[models.ProviderAnthropic].Disabled)
	require.Empty(t, base.Providers[models.ProviderAnthropic].APIKey)

	data, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".kotha.json"))
	require.NoError(t, err)
	var onDisk Config
	require.NoError(t, json.Unmarshal(data, &onDisk))
	require.True(t, onDisk.Providers[models.ProviderAnthropic].Disabled)
}

func TestSetProviderEnableWithoutKeyMarksDisabled(t *testing.T) {
	isolateEnv(t)
	base := &Config{
		WorkingDir: t.TempDir(),
		Providers:  map[models.ModelProvider]Provider{},
		Agents:     map[AgentName]Agent{},
	}
	setCfgWithHome(t, base)

	p := GetDefaultProvider()
	// No env credentials available, so enabling without a key should mark
	// the provider disabled rather than leaving it half-configured.
	require.NoError(t, p.SetProvider(models.ProviderOpenAI, true, ""))
	require.True(t, base.Providers[models.ProviderOpenAI].Disabled)
}

func TestSetProviderEnableWithEnvCredentials(t *testing.T) {
	isolateEnv(t)
	t.Setenv("OPENAI_API_KEY", "env-key")
	base := &Config{
		WorkingDir: t.TempDir(),
		Providers:  map[models.ModelProvider]Provider{},
		Agents:     map[AgentName]Agent{},
	}
	setCfgWithHome(t, base)

	p := GetDefaultProvider()
	require.NoError(t, p.SetProvider(models.ProviderOpenAI, true, ""))
	require.False(t, base.Providers[models.ProviderOpenAI].Disabled)
}

func TestSetProviderNilConfigErrors(t *testing.T) {
	p := NewDefaultProvider(nil)
	require.Error(t, p.SetProvider(models.ProviderAnthropic, true, "k"))
}

func TestDefaultModelForAgentRespectsPersistedProviders(t *testing.T) {
	isolateEnv(t)
	base := &Config{
		WorkingDir: t.TempDir(),
		Providers: map[models.ModelProvider]Provider{
			models.ProviderOpenAI: {APIKey: "k"},
		},
		Agents: map[AgentName]Agent{
			AgentCoder: {Model: models.Claude4Sonnet},
		},
	}
	setCfgWithHome(t, base)

	// With only OpenAI configured, the default model should switch to GPT-4.1
	// even though the current agent model is a Claude model.
	got := DefaultModelForAgent(AgentCoder)
	require.Equal(t, models.GPT41, got)

	// Disabling OpenAI leaves no usable provider, so the default reverts to
	// whatever was previously configured.
	require.NoError(t, GetDefaultProvider().SetProvider(models.ProviderOpenAI, false, ""))
	got = DefaultModelForAgent(AgentCoder)
	require.Equal(t, models.Claude4Sonnet, got)
}

func TestProviderEnabled(t *testing.T) {
	isolateEnv(t)
	base := &Config{
		WorkingDir: t.TempDir(),
		Providers: map[models.ModelProvider]Provider{
			models.ProviderAnthropic: {APIKey: "k"},
			models.ProviderOpenAI:    {Disabled: true},
		},
		Agents: map[AgentName]Agent{},
	}
	setCfgWithHome(t, base)

	require.True(t, providerEnabled(models.ProviderAnthropic))
	require.False(t, providerEnabled(models.ProviderOpenAI))

	// An explicitly disabled provider stays disabled even when env
	// credentials become available.
	t.Setenv("OPENAI_API_KEY", "env")
	require.False(t, providerEnabled(models.ProviderOpenAI))

	// A provider with no persisted entry but env credentials is enabled.
	t.Setenv("GROQ_API_KEY", "env")
	require.True(t, providerEnabled(models.ProviderGROQ))
}
