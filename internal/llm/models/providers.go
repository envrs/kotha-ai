package models

import "os"

// AllProviders is the canonical, display-ordered list of every supported
// LLM provider. It is used by the provider connect dialog to enumerate the
// providers a user can enable/disable and configure.
var AllProviders = []ModelProvider{
	ProviderCopilot,
	ProviderAnthropic,
	ProviderOpenAI,
	ProviderGemini,
	ProviderGROQ,
	ProviderOpenRouter,
	ProviderXAI,
	ProviderBedrock,
	ProviderAzure,
	ProviderVertexAI,
	ProviderLocal,
}

// ProviderDisplayName maps a provider to a human-friendly label shown in the UI.
var ProviderDisplayName = map[ModelProvider]string{
	ProviderCopilot:    "GitHub Copilot",
	ProviderAnthropic:  "Anthropic",
	ProviderOpenAI:     "OpenAI",
	ProviderGemini:     "Google Gemini",
	ProviderGROQ:       "Groq",
	ProviderOpenRouter: "OpenRouter",
	ProviderXAI:        "xAI",
	ProviderBedrock:    "AWS Bedrock",
	ProviderAzure:      "Azure OpenAI",
	ProviderVertexAI:   "Google Vertex AI",
	ProviderLocal:      "Local (LM Studio)",
}

// ProviderEnvVar maps a provider to the environment variable that supplies its
// API key/credentials. Empty means the provider uses non-token credentials
// (e.g. AWS SDK chain, Google ADC).
var ProviderEnvVar = map[ModelProvider]string{
	ProviderCopilot:    "GITHUB_TOKEN",
	ProviderAnthropic:  "ANTHROPIC_API_KEY",
	ProviderOpenAI:     "OPENAI_API_KEY",
	ProviderGemini:     "GEMINI_API_KEY",
	ProviderGROQ:       "GROQ_API_KEY",
	ProviderOpenRouter: "OPENROUTER_API_KEY",
	ProviderXAI:        "XAI_API_KEY",
	ProviderBedrock:    "",
	ProviderAzure:      "AZURE_OPENAI_API_KEY",
	ProviderVertexAI:   "",
	ProviderLocal:      "LOCAL_ENDPOINT",
}

// ProviderHasCredentials reports whether the provider has usable credentials
// available in the environment right now.
func ProviderHasCredentials(p ModelProvider) bool {
	if p == ProviderLocal {
		return os.Getenv("LOCAL_ENDPOINT") != ""
	}
	if p == ProviderBedrock {
		return hasAWSCredentials()
	}
	if p == ProviderVertexAI {
		return hasVertexAICredentials()
	}
	env := ProviderEnvVar[p]
	return env != "" && os.Getenv(env) != ""
}

// hasAWSCredentials reports whether AWS credentials are available in the
// environment. It mirrors config.hasAWSCredentials so the models package does
// not depend on the config package.
func hasAWSCredentials() bool {
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return true
	}
	if os.Getenv("AWS_PROFILE") != "" || os.Getenv("AWS_DEFAULT_PROFILE") != "" {
		return true
	}
	if os.Getenv("AWS_REGION") != "" || os.Getenv("AWS_DEFAULT_REGION") != "" {
		return true
	}
	if os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI") != "" ||
		os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI") != "" {
		return true
	}
	return false
}

// hasVertexAICredentials reports whether Google Cloud / Vertex AI credentials
// are available in the environment.
func hasVertexAICredentials() bool {
	if os.Getenv("VERTEXAI_PROJECT") != "" && os.Getenv("VERTEXAI_LOCATION") != "" {
		return true
	}
	if os.Getenv("GOOGLE_CLOUD_PROJECT") != "" &&
		(os.Getenv("GOOGLE_CLOUD_REGION") != "" || os.Getenv("GOOGLE_CLOUD_LOCATION") != "") {
		return true
	}
	return false
}
