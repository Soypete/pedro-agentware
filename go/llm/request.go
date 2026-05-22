package llm

type MessageType string

const (
	MessageTypeSystemPrompt      MessageType = "system_prompt"
	MessageTypeUserInput         MessageType = "user_input"
	MessageTypeToolCall          MessageType = "tool_call"
	MessageTypeToolResult        MessageType = "tool_result"
	MessageTypeReasoning         MessageType = "reasoning"
	MessageTypeTextResponse      MessageType = "text_response"
	MessageTypeStepNudge         MessageType = "step_nudge"
	MessageTypePrerequisiteNudge MessageType = "prerequisite_nudge"
	MessageTypeRetryNudge        MessageType = "retry_nudge"
	MessageTypeContextWarning    MessageType = "context_warning"
	MessageTypeSummary           MessageType = "summary"
)

type MessageMeta struct {
	Type          MessageType
	StepIndex     *int
	OriginalType  *MessageType
	TokenEstimate *int
}

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
	Meta       MessageMeta
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
