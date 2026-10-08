package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// isolateEnv clears all provider/credential env vars and points HOME at a
// temp dir so config and token discovery cannot see the real machine.
func isolateEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GEMINI_API_KEY", "GROQ_API_KEY",
		"OPENROUTER_API_KEY", "XAI_API_KEY", "AZURE_OPENAI_ENDPOINT",
		"AZURE_OPENAI_API_KEY", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY",
		"AWS_PROFILE", "AWS_DEFAULT_PROFILE", "AWS_REGION", "AWS_DEFAULT_REGION",
		"AWS_CONTAINER_CREDENTIALS_RELATIVE_URI", "AWS_CONTAINER_CREDENTIALS_FULL_URI",
		"VERTEXAI_PROJECT", "VERTEXAI_LOCATION", "GOOGLE_CLOUD_PROJECT",
		"GOOGLE_CLOUD_REGION", "GOOGLE_CLOUD_LOCATION", "GITHUB_TOKEN",
		"KOTHA_DEV_DEBUG", "SHELL",
	} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("LOCALAPPDATA", dir)
}

// setCfg installs a config for the duration of the test and restores
// the previous global.
func setCfg(t *testing.T, c *Config) *Config {
	t.Helper()
	old := cfg
	cfg = c
	t.Cleanup(func() { cfg = old })
	return c
}

func TestReadConfig(t *testing.T) {
	require.NoError(t, readConfig(nil))
	require.NoError(t, readConfig(viper.ConfigFileNotFoundError{}))
	err := readConfig(errors.New("boom"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read config")
}

func TestApplyDefaultValuesMCPType(t *testing.T) {
	c := setCfg(t, &Config{MCPServers: map[string]MCPServer{
		"srv": {Command: "x"},
		"web": {Type: MCPSse, URL: "http://x"},
	}})
	applyDefaultValues()
	require.Equal(t, MCPStdio, c.MCPServers["srv"].Type)
	require.Equal(t, MCPSse, c.MCPServers["web"].Type)
}

func TestSetDefaultsShellAndDebug(t *testing.T) {
	t.Cleanup(viper.Reset)
	t.Setenv("SHELL", "/bin/zsh")
	setDefaults(false)
	require.Equal(t, "/bin/zsh", viper.GetString("shell.path"))
	require.Equal(t, defaultDataDirectory, viper.GetString("data.directory"))
	require.Equal(t, "kotha", viper.GetString("tui.theme"))
	require.False(t, viper.GetBool("debug"))

	t.Setenv("SHELL", "")
	setDefaults(false)
	require.Equal(t, "/bin/bash", viper.GetString("shell.path"))

	viper.Reset()
	setDefaults(true)
	require.True(t, viper.GetBool("debug"))
}

func TestMergeLocalConfig(t *testing.T) {
	t.Cleanup(viper.Reset)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".kotha.json"),
		[]byte(`{"tui":{"theme":"custom"}}`), 0o644))
	mergeLocalConfig(dir)
	require.Equal(t, "custom", viper.GetString("tui.theme"))
}

func TestGetProviderAPIKey(t *testing.T) {
	isolateEnv(t)
	cases := []struct {
		provider models.ModelProvider
		env      string
		want     string
	}{
		{models.ProviderAnthropic, "ANTHROPIC_API_KEY", "k-a"},
		{models.ProviderOpenAI, "OPENAI_API_KEY", "k-o"},
		{models.ProviderGemini, "GEMINI_API_KEY", "k-g"},
		{models.ProviderGROQ, "GROQ_API_KEY", "k-q"},
		{models.ProviderAzure, "AZURE_OPENAI_API_KEY", "k-z"},
		{models.ProviderOpenRouter, "OPENROUTER_API_KEY", "k-r"},
	}
	for _, tc := range cases {
		t.Setenv(tc.env, tc.want)
		require.Equal(t, tc.want, getProviderAPIKey(tc.provider), tc.provider)
	}

	require.Empty(t, getProviderAPIKey(models.ModelProvider("unknown")))

	require.Empty(t, getProviderAPIKey(models.ProviderBedrock))
	t.Setenv("AWS_PROFILE", "dev")
	require.Equal(t, "aws-credentials-available", getProviderAPIKey(models.ProviderBedrock))

	require.Empty(t, getProviderAPIKey(models.ProviderVertexAI))
	t.Setenv("VERTEXAI_PROJECT", "p")
	t.Setenv("VERTEXAI_LOCATION", "us")
	require.Equal(t, "vertex-ai-credentials-available", getProviderAPIKey(models.ProviderVertexAI))
}

func TestHasAWSCredentials(t *testing.T) {
	isolateEnv(t)
	require.False(t, hasAWSCredentials())

	cases := map[string]map[string]string{
		"explicit":      {"AWS_ACCESS_KEY_ID": "id", "AWS_SECRET_ACCESS_KEY": "sec"},
		"profile":       {"AWS_PROFILE": "dev"},
		"defaultprof":   {"AWS_DEFAULT_PROFILE": "dev"},
		"region":        {"AWS_REGION": "us-east-1"},
		"defaultregion": {"AWS_DEFAULT_REGION": "us-east-1"},
		"container":     {"AWS_CONTAINER_CREDENTIALS_RELATIVE_URI": "/x"},
		"fullcontainer": {"AWS_CONTAINER_CREDENTIALS_FULL_URI": "http://x"},
	}
	for name, envs := range cases {
		t.Run(name, func(t *testing.T) {
			isolateEnv(t)
			for k, v := range envs {
				t.Setenv(k, v)
			}
			require.True(t, hasAWSCredentials())
		})
	}
}

func TestHasVertexAICredentials(t *testing.T) {
	isolateEnv(t)
	require.False(t, hasVertexAICredentials())

	t.Setenv("VERTEXAI_PROJECT", "p")
	t.Setenv("VERTEXAI_LOCATION", "us-central1")
	require.True(t, hasVertexAICredentials())

	isolateEnv(t)
	t.Setenv("GOOGLE_CLOUD_PROJECT", "p")
	t.Setenv("GOOGLE_CLOUD_REGION", "us-central1")
	require.True(t, hasVertexAICredentials())

	isolateEnv(t)
	t.Setenv("GOOGLE_CLOUD_PROJECT", "p")
	t.Setenv("GOOGLE_CLOUD_LOCATION", "us-central1")
	require.True(t, hasVertexAICredentials())

	isolateEnv(t)
	t.Setenv("GOOGLE_CLOUD_PROJECT", "p")
	require.False(t, hasVertexAICredentials())
}

// envOnly returns a config with a usable Anthropic provider and clears
// everything else so validateAgent falls back deterministically.
func validateFixture(t *testing.T) *Config {
	t.Helper()
	isolateEnv(t)
	return setCfg(t, &Config{
		MCPServers: map[string]MCPServer{},
		Providers:  map[models.ModelProvider]Provider{},
		LSP:        map[string]LSPConfig{},
		Agents:     map[AgentName]Agent{},
	})
}

func TestValidateAgentUnsupportedModelNoProvider(t *testing.T) {
	validateFixture(t)
	err := validateAgent(cfg, AgentCoder, Agent{Model: "does-not-exist", MaxTokens: 100})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no valid provider available")
}

func TestValidateAgentUnsupportedModelFallsBack(t *testing.T) {
	validateFixture(t)
	t.Setenv("ANTHROPIC_API_KEY", "key")
	err := validateAgent(cfg, AgentCoder, Agent{Model: "does-not-exist", MaxTokens: 100})
	require.NoError(t, err)
	require.Equal(t, models.Claude37Sonnet, cfg.Agents[AgentCoder].Model)
	require.Equal(t, int64(5000), cfg.Agents[AgentCoder].MaxTokens)

	err = validateAgent(cfg, AgentTitle, Agent{Model: "does-not-exist", MaxTokens: 100})
	require.NoError(t, err)
	require.Equal(t, int64(80), cfg.Agents[AgentTitle].MaxTokens)
}

func TestValidateAgentAddsProviderFromEnv(t *testing.T) {
	validateFixture(t)
	t.Setenv("ANTHROPIC_API_KEY", "env-key")
	err := validateAgent(cfg, AgentCoder, Agent{Model: models.Claude4Sonnet, MaxTokens: 100})
	require.NoError(t, err)
	require.Equal(t, "env-key", cfg.Providers[models.ProviderAnthropic].APIKey)
}

func TestValidateAgentDisabledProviderReverts(t *testing.T) {
	validateFixture(t)
	cfg.Providers[models.ProviderAnthropic] = Provider{Disabled: true}
	t.Setenv("ANTHROPIC_API_KEY", "env-key")
	err := validateAgent(cfg, AgentCoder, Agent{Model: models.Claude4Sonnet, MaxTokens: 100})
	require.NoError(t, err)
	require.Equal(t, models.Claude37Sonnet, cfg.Agents[AgentCoder].Model)
}

func TestValidateAgentProviderWithoutKeyFails(t *testing.T) {
	validateFixture(t)
	cfg.Providers[models.ProviderAnthropic] = Provider{APIKey: ""}
	err := validateAgent(cfg, AgentCoder, Agent{Model: models.Claude4Sonnet, MaxTokens: 100})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no valid provider available")
}

func TestValidateAgentMaxTokens(t *testing.T) {
	t.Run("zero uses model default", func(t *testing.T) {
		validateFixture(t)
		cfg.Providers[models.ProviderAnthropic] = Provider{APIKey: "k"}
		err := validateAgent(cfg, AgentCoder, Agent{Model: models.Claude4Sonnet, MaxTokens: 0})
		require.NoError(t, err)
		require.Equal(t, int64(50000), cfg.Agents[AgentCoder].MaxTokens)
	})

	t.Run("clamped to half context window", func(t *testing.T) {
		validateFixture(t)
		cfg.Providers[models.ProviderAnthropic] = Provider{APIKey: "k"}
		err := validateAgent(cfg, AgentCoder, Agent{Model: models.Claude4Sonnet, MaxTokens: 300000})
		require.NoError(t, err)
		require.Equal(t, int64(100000), cfg.Agents[AgentCoder].MaxTokens)
	})

	t.Run("valid value kept", func(t *testing.T) {
		validateFixture(t)
		cfg.Providers[models.ProviderAnthropic] = Provider{APIKey: "k"}
		err := validateAgent(cfg, AgentCoder, Agent{Model: models.Claude4Sonnet, MaxTokens: 1234})
		require.NoError(t, err)
		require.NotContains(t, cfg.Agents, AgentCoder)
	})
}

func TestValidateAgentReasoningEffort(t *testing.T) {
	t.Run("openai reasoning model gets default effort", func(t *testing.T) {
		validateFixture(t)
		cfg.Providers[models.ProviderOpenAI] = Provider{APIKey: "k"}
		err := validateAgent(cfg, AgentCoder, Agent{Model: models.O1, MaxTokens: 100})
		require.NoError(t, err)
		require.Equal(t, "medium", cfg.Agents[AgentCoder].ReasoningEffort)
	})

	t.Run("invalid effort normalized", func(t *testing.T) {
		validateFixture(t)
		cfg.Providers[models.ProviderOpenAI] = Provider{APIKey: "k"}
		err := validateAgent(cfg, AgentCoder, Agent{Model: models.O1, MaxTokens: 100, ReasoningEffort: "bogus"})
		require.NoError(t, err)
		require.Equal(t, "medium", cfg.Agents[AgentCoder].ReasoningEffort)
	})

	t.Run("valid effort preserved", func(t *testing.T) {
		validateFixture(t)
		cfg.Providers[models.ProviderOpenAI] = Provider{APIKey: "k"}
		err := validateAgent(cfg, AgentCoder, Agent{Model: models.O1, MaxTokens: 100, ReasoningEffort: "high"})
		require.NoError(t, err)
		require.NotContains(t, cfg.Agents, AgentCoder)
	})

	t.Run("effort cleared for non-reasoning model", func(t *testing.T) {
		validateFixture(t)
		cfg.Providers[models.ProviderOpenAI] = Provider{APIKey: "k"}
		err := validateAgent(cfg, AgentCoder, Agent{Model: models.GPT41, MaxTokens: 100, ReasoningEffort: "high"})
		require.NoError(t, err)
		require.Empty(t, cfg.Agents[AgentCoder].ReasoningEffort)
	})

	t.Run("local provider gets default effort", func(t *testing.T) {
		validateFixture(t)
		cfg.Providers[models.ProviderLocal] = Provider{APIKey: "k"}
		const localID models.ModelID = "zz-test-local"
		models.SupportedModels[localID] = models.Model{
			ID:               localID,
			Provider:         models.ProviderLocal,
			ContextWindow:    8000,
			DefaultMaxTokens: 1000,
		}
		t.Cleanup(func() { delete(models.SupportedModels, localID) })

		err := validateAgent(cfg, AgentCoder, Agent{Model: localID, MaxTokens: 100})
		require.NoError(t, err)
		require.Equal(t, "medium", cfg.Agents[AgentCoder].ReasoningEffort)
	})
}

func TestValidate(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		setCfg(t, nil)
		require.Error(t, Validate())
	})

	t.Run("marks provider and lsp disabled", func(t *testing.T) {
		validateFixture(t)
		prov := models.ModelProvider("someprov")
		cfg.Providers[prov] = Provider{}
		cfg.LSP["go"] = LSPConfig{}
		require.NoError(t, Validate())
		require.True(t, cfg.Providers[prov].Disabled)
		require.True(t, cfg.LSP["go"].Disabled)
	})

	t.Run("bad agent fails", func(t *testing.T) {
		validateFixture(t)
		cfg.Agents[AgentCoder] = Agent{Model: "missing-model", MaxTokens: 1}
		require.Error(t, Validate())
	})
}

func TestLoadGitHubToken(t *testing.T) {
	t.Run("env wins", func(t *testing.T) {
		isolateEnv(t)
		t.Setenv("GITHUB_TOKEN", "gho_env")
		tok, err := LoadGitHubToken()
		require.NoError(t, err)
		require.Equal(t, "gho_env", tok)
	})

	t.Run("hosts.json file", func(t *testing.T) {
		isolateEnv(t)
		dir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "github-copilot")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "hosts.json"),
			[]byte(`{"github.com":{"oauth_token":"gho_file"}}`), 0o644))
		tok, err := LoadGitHubToken()
		require.NoError(t, err)
		require.Equal(t, "gho_file", tok)
	})

	t.Run("apps.json file", func(t *testing.T) {
		isolateEnv(t)
		dir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "github-copilot")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "apps.json"),
			[]byte(`{"github.com":{"oauth_token":"gho_apps"}}`), 0o644))
		tok, err := LoadGitHubToken()
		require.NoError(t, err)
		require.Equal(t, "gho_apps", tok)
	})

	t.Run("not found", func(t *testing.T) {
		isolateEnv(t)
		_, err := LoadGitHubToken()
		require.Error(t, err)
	})

	t.Run("malformed file skipped", func(t *testing.T) {
		isolateEnv(t)
		dir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "github-copilot")
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "hosts.json"),
			[]byte(`{not json`), 0o644))
		_, err := LoadGitHubToken()
		require.Error(t, err)
	})
}

func TestSetDefaultModelForAgentBranches(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want models.ModelID
	}{
		{"anthropic", map[string]string{"ANTHROPIC_API_KEY": "k"}, models.Claude37Sonnet},
		{"openai", map[string]string{"OPENAI_API_KEY": "k"}, models.GPT41},
		{"openrouter", map[string]string{"OPENROUTER_API_KEY": "k"}, models.OpenRouterClaude37Sonnet},
		{"gemini", map[string]string{"GEMINI_API_KEY": "k"}, models.Gemini25},
		{"groq", map[string]string{"GROQ_API_KEY": "k"}, models.QWENQwq},
		{"aws", map[string]string{"AWS_PROFILE": "p"}, models.BedrockClaude37Sonnet},
		{"vertex", map[string]string{"VERTEXAI_PROJECT": "p", "VERTEXAI_LOCATION": "us"}, models.VertexAIGemini25},
		{"copilot", map[string]string{"GITHUB_TOKEN": "t"}, models.CopilotGPT4o},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			validateFixture(t)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			require.True(t, setDefaultModelForAgent(AgentCoder))
			require.Equal(t, tc.want, cfg.Agents[AgentCoder].Model)
			require.Equal(t, int64(5000), cfg.Agents[AgentCoder].MaxTokens)

			require.True(t, setDefaultModelForAgent(AgentTitle))
			require.Equal(t, int64(80), cfg.Agents[AgentTitle].MaxTokens)
		})
	}

	t.Run("none available", func(t *testing.T) {
		validateFixture(t)
		require.False(t, setDefaultModelForAgent(AgentCoder))
	})
}

func TestLoadAlreadyLoaded(t *testing.T) {
	isolateEnv(t)
	want := &Config{WorkingDir: "/already"}
	setCfg(t, want)
	old := defaultProviderInstance
	SetDefaultProvider(NewDefaultProvider(want))
	t.Cleanup(func() { defaultProviderInstance = old })

	p, err := Load("/other", false)
	require.NoError(t, err)
	require.Equal(t, want, p.Get())
}

func TestUpdateAgentModelAndTheme(t *testing.T) {
	isolateEnv(t)
	t.Setenv("ANTHROPIC_API_KEY", "k")
	base := &Config{
		WorkingDir: t.TempDir(),
		Providers:  map[models.ModelProvider]Provider{},
		Agents: map[AgentName]Agent{
			AgentCoder: {Model: models.Claude4Sonnet, MaxTokens: 999, ReasoningEffort: "high"},
		},
	}
	setCfg(t, base)
	old := defaultProviderInstance
	SetDefaultProvider(NewDefaultProvider(base))
	t.Cleanup(func() { defaultProviderInstance = old })

	t.Run("update agent model", func(t *testing.T) {
		require.NoError(t, UpdateAgentModel(AgentCoder, models.Claude4Sonnet))
		require.Equal(t, models.Claude4Sonnet, base.Agents[AgentCoder].Model)
		require.Equal(t, int64(50000), base.Agents[AgentCoder].MaxTokens)
		require.Equal(t, "high", base.Agents[AgentCoder].ReasoningEffort)

		data, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".kotha.json"))
		require.NoError(t, err)
		var onDisk Config
		require.NoError(t, json.Unmarshal(data, &onDisk))
		require.Equal(t, models.Claude4Sonnet, onDisk.Agents[AgentCoder].Model)
	})

	t.Run("unsupported model rejected", func(t *testing.T) {
		err := UpdateAgentModel(AgentCoder, "nope")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not supported")
		require.Equal(t, models.Claude4Sonnet, base.Agents[AgentCoder].Model)
	})

	t.Run("validation failure rolls back", func(t *testing.T) {
		isolateEnv(t)
		err := UpdateAgentModel(AgentCoder, models.BedrockClaude37Sonnet)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to validate agent config")
		require.Equal(t, models.Claude4Sonnet, base.Agents[AgentCoder].Model)
	})

	t.Run("update theme", func(t *testing.T) {
		require.NoError(t, UpdateTheme("dracula"))
		require.Equal(t, "dracula", base.TUI.Theme)
		data, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".kotha.json"))
		require.NoError(t, err)
		require.Contains(t, string(data), "dracula")
	})

	t.Run("existing file updated", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "cfg.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"tui":{"theme":"old"}}`), 0o644))
		viper.SetConfigFile(path)
		t.Cleanup(viper.Reset)

		require.NoError(t, UpdateTheme("new"))
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Contains(t, string(data), "new")
	})

	t.Run("invalid config file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "cfg.json")
		require.NoError(t, os.WriteFile(path, []byte(`{broken`), 0o644))
		viper.SetConfigFile(path)
		t.Cleanup(viper.Reset)

		err := UpdateTheme("x")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse config file")
	})

	t.Run("unreadable config file", func(t *testing.T) {
		viper.SetConfigFile(filepath.Join(t.TempDir(), "missing.json"))
		t.Cleanup(viper.Reset)

		err := UpdateTheme("x")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read config file")
	})
}

func TestDefaultProviderNilErrors(t *testing.T) {
	p := NewDefaultProvider(nil)
	require.Error(t, p.UpdateAgentModel(AgentCoder, models.Claude4Sonnet))
	require.Error(t, p.UpdateTheme("x"))
}
