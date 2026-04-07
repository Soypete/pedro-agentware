package toolformat

import "strings"

func GetFormatter(modelName string) ToolFormatter {
	lower := strings.ToLower(modelName)
	switch {
	case strings.Contains(lower, "qwen"):
		return &QwenFormatter{}
	case strings.Contains(lower, "llama"):
		return &LlamaFormatter{}
	case strings.Contains(lower, "mistral"):
		return &MistralFormatter{}
	default:
		return &GenericFormatter{}
	}
}
