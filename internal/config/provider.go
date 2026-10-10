package config

import (
	"fmt"
	"os"

	"github.com/kothagpt/kotha/internal/llm/models"
)

// Provider is the interface for accessing application configuration.
// It abstracts the global config singleton and enables dependency injection.
type ConfigProvider interface {
	Get() *Config
	WorkingDirectory() string
	UpdateAgentModel(agentName AgentName, modelID models.ModelID) error
	UpdateTheme(themeName string) error
	// SetProvider enables or disables a provider and optionally sets its API key.
	// It persists the change to the config file and updates the in-memory config.
	SetProvider(provider models.ModelProvider, enabled bool, apiKey string) error
}

// defaultProvider is the default implementation of Provider that wraps a *Config.
type defaultProvider struct {
	cfg *Config
}

// NewDefaultProvider creates a new Provider from a loaded Config.
func NewDefaultProvider(cfg *Config) ConfigProvider {
	return &defaultProvider{cfg: cfg}
}

func (p *defaultProvider) Get() *Config {
	return p.cfg
}

func (p *defaultProvider) WorkingDirectory() string {
	if p.cfg == nil {
		panic("config not loaded")
	}
	return p.cfg.WorkingDir
}

func (p *defaultProvider) UpdateAgentModel(agentName AgentName, modelID models.ModelID) error {
	if p.cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	existingAgentCfg := p.cfg.Agents[agentName]

	model, ok := models.SupportedModels[modelID]
	if !ok {
		return fmt.Errorf("model %s not supported", modelID)
	}

	maxTokens := existingAgentCfg.MaxTokens
	if model.DefaultMaxTokens > 0 {
		maxTokens = model.DefaultMaxTokens
	}

	newAgentCfg := Agent{
		Model:           modelID,
		MaxTokens:       maxTokens,
		ReasoningEffort: existingAgentCfg.ReasoningEffort,
	}
	p.cfg.Agents[agentName] = newAgentCfg

	if err := validateAgent(p.cfg, agentName, newAgentCfg); err != nil {
		p.cfg.Agents[agentName] = existingAgentCfg
		return fmt.Errorf("failed to validate agent config: %w", err)
	}

	return updateCfgFile(func(config *Config) {
		if config.Agents == nil {
			config.Agents = make(map[AgentName]Agent)
		}
		config.Agents[agentName] = newAgentCfg
	})
}

func (p *defaultProvider) UpdateTheme(themeName string) error {
	if p.cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	p.cfg.TUI.Theme = themeName

	return updateCfgFile(func(config *Config) {
		config.TUI.Theme = themeName
	})
}

// SetProvider enables or disables a provider and optionally sets its API key.
// When a provider is enabled without an explicit API key and without
// credentials available in the environment, it is marked disabled to stay
// consistent with Validate().
func (p *defaultProvider) SetProvider(provider models.ModelProvider, enabled bool, apiKey string) error {
	if p.cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	if p.cfg.Providers == nil {
		p.cfg.Providers = make(map[models.ModelProvider]Provider)
	}

	existing := p.cfg.Providers[provider]
	if !enabled {
		// Disabling: persist the disabled state and clear any stored key.
		existing.Disabled = true
		existing.APIKey = ""
		p.cfg.Providers[provider] = existing

		return updateCfgFile(func(config *Config) {
			if config.Providers == nil {
				config.Providers = make(map[models.ModelProvider]Provider)
			}
			config.Providers[provider] = Provider{Disabled: true}
		})
	}

	// Enabling.
	newCfg := Provider{
		APIKey:   apiKey,
		Disabled: false,
	}
	if newCfg.APIKey == "" && !providerHasCredentials(provider) {
		// No key provided and no env credentials: mark disabled so the
		// provider does not surface in the model dialog until configured.
		newCfg.Disabled = true
	}
	p.cfg.Providers[provider] = newCfg

	return updateCfgFile(func(config *Config) {
		if config.Providers == nil {
			config.Providers = make(map[models.ModelProvider]Provider)
		}
		config.Providers[provider] = newCfg
	})
}

// providerHasCredentials reports whether the provider has usable credentials
// available in the environment right now. It mirrors the logic in
// config.getProviderAPIKey / setProviderDefaults so the provider package does
// not need to import config.
func providerHasCredentials(provider models.ModelProvider) bool {
	switch provider {
	case models.ProviderBedrock:
		return envHas("AWS_ACCESS_KEY_ID") && envHas("AWS_SECRET_ACCESS_KEY") ||
			envHas("AWS_PROFILE") || envHas("AWS_DEFAULT_PROFILE") ||
			envHas("AWS_REGION") || envHas("AWS_DEFAULT_REGION") ||
			envHas("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI") ||
			envHas("AWS_CONTAINER_CREDENTIALS_FULL_URI")
	case models.ProviderVertexAI:
		return envHas("VERTEXAI_PROJECT") && envHas("VERTEXAI_LOCATION") ||
			envHas("GOOGLE_CLOUD_PROJECT") && (envHas("GOOGLE_CLOUD_REGION") || envHas("GOOGLE_CLOUD_LOCATION"))
	case models.ProviderLocal:
		return envHas("LOCAL_ENDPOINT")
	default:
		return envHas(models.ProviderEnvVar[provider])
	}
}

func envHas(key string) bool {
	return key != "" && os.Getenv(key) != ""
}

// defaultProviderInstance is the global default provider instance.
// It is set by Load() and used by backward-compatible package-level functions.
var defaultProviderInstance ConfigProvider

// SetDefaultProvider sets the global default provider instance.
// This is called by Load() and can be overridden in tests.
func SetDefaultProvider(p ConfigProvider) {
	defaultProviderInstance = p
}

// GetDefaultProvider returns the global default provider instance.
func GetDefaultProvider() ConfigProvider {
	if defaultProviderInstance == nil {
		panic("config not loaded")
	}
	return defaultProviderInstance
}
