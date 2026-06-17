package proxy

import "time"

type BackendType string

const (
	BackendTypeLlamaServer BackendType = "llamaserver"
	BackendTypeOllama      BackendType = "ollama"
	BackendTypeAnthropic   BackendType = "anthropic"
)

type ProxyConfig struct {
	BackendURL    string
	BackendType   BackendType
	ListenAddr    string
	MaxRetries    int
	KeepRecent    int
	ContextWindow int
	Timeout       time.Duration
	APIKey        string
	Model         string
}

func DefaultConfig() *ProxyConfig {
	return &ProxyConfig{
		BackendURL:    "http://localhost:8080",
		BackendType:   BackendTypeLlamaServer,
		ListenAddr:    "127.0.0.1:8081",
		MaxRetries:    3,
		KeepRecent:    10,
		ContextWindow: 32768,
		Timeout:       5 * time.Minute,
		Model:         "default",
	}
}
