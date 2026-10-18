package config

import (
	"fmt"

	"github.com/kothagpt/kotha/ai/internal/llm/models"
)

// Provider is the interface for accessing application configuration.
// It abstracts the global config singleton and enables dependency injection.
type ConfigProvider interface {
	Get() *Config
	WorkingDirectory() string
	UpdateAgentModel(agentName AgentName, modelID models.ModelID) error
	UpdateTheme(themeName string) error
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
