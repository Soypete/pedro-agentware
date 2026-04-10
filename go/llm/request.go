package llm

// Request is the input to a completion.
type Request struct {
	Messages    []Message
	Tools       []ToolDefinition
	Temperature float64
	MaxTokens   int
	Stop        []string
}

// Message is a single turn in a conversation.
type Message struct {
	Role       Role
	Content    string
	ToolCallID string
	ToolCalls  []ToolCall
}

// Role represents the role of a message sender.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolDefinition is the schema for a tool, used in native tool calling requests.
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]any
}
