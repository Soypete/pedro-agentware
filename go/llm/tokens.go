package llm

import "fmt"

// EstimateTokens estimates the number of tokens in a message.
// This is a rough approximation; actual tokenizers vary by model.
func EstimateTokens(text string) int {
	return len(text) / 4
}

// EstimateMessagesTokens estimates tokens for a slice of messages.
func EstimateMessagesTokens(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += 4 // overhead per message
		total += EstimateTokens(m.Content)
		for _, tc := range m.ToolCalls {
			total += EstimateTokens(tc.Name)
			for _, v := range tc.Args {
				total += EstimateTokens(fmt.Sprintf("%v", v))
			}
		}
	}
	return total
}

// GetModelContextWindow returns the context window for known models.
// Returns 0 if unknown.
func GetModelContextWindow(modelName string) int {
	switch {
	case contains(modelName, "gpt-4o"):
		return 128000
	case contains(modelName, "gpt-4-turbo"):
		return 128000
	case contains(modelName, "gpt-4"):
		return 8192
	case contains(modelName, "gpt-3.5-turbo"):
		return 16385
	case contains(modelName, "claude-3-opus"):
		return 200000
	case contains(modelName, "claude-3-sonnet"):
		return 200000
	case contains(modelName, "claude-3-haiku"):
		return 200000
	case contains(modelName, "qwen"):
		return 32768
	case contains(modelName, "llama"):
		return 8192
	case contains(modelName, "mistral"):
		return 32768
	default:
		return 4096
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
