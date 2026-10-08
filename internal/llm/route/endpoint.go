package route

// Endpoint presets. Auth is intentionally left to the caller so secrets
// never live in code; use FromEnv/APIKey/Bearer to attach credentials.

func MockEndpoint(model string) Endpoint {
	return Endpoint{Name: "mock", Provider: "mock", Model: model}
}

func OpenAIEndpoint(model, url string, auth Credentials) Endpoint {
	return Endpoint{Name: "openai", Provider: "openai", Model: model, URL: url, Auth: auth}
}

func AnthropicEndpoint(model, url string, auth Credentials) Endpoint {
	return Endpoint{Name: "anthropic", Provider: "anthropic", Model: model, URL: url, Auth: auth}
}

func LocalEndpoint(model, url string) Endpoint {
	return Endpoint{Name: "local", Provider: "local", Model: model, URL: url, Auth: NoAuth()}
}
