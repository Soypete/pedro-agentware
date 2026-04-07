package llm

import "time"

// Config holds configuration for creating a Backend.
type Config struct {
	BaseURL       string
	APIKey        string
	Model         string
	ContextWindow int
	Timeout       time.Duration
}

// NewBackend creates a new Backend based on the configuration.
// Currently supports OpenAI-compatible servers.
func NewBackend(config Config) (Backend, error) {
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.ContextWindow == 0 {
		config.ContextWindow = GetModelContextWindow(config.Model)
	}

	return NewServerBackend(ServerConfig(config)), nil
}
