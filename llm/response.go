package llm

// Response is the output from a completion.
type Response struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason string
	UsageTokens  TokenUsage
}

// ToolCall is a structured tool invocation from the LLM.
type ToolCall struct {
	ID   string
	Name string
	Args map[string]any
}

// TokenUsage tracks prompt and completion token counts.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}
