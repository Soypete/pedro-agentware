package format

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/pedro/agent-middleware/types"
)

type ModelFamily string

const (
	ModelFamilyLlama3  ModelFamily = "llama3"
	ModelFamilyQwen    ModelFamily = "qwen"
	ModelFamilyMistral ModelFamily = "mistral"
	ModelFamilyClaude  ModelFamily = "claude"
	ModelFamilyOpenAI  ModelFamily = "openai"
	ModelFamilyGLM4    ModelFamily = "glm4"
)

type ToolFormatter interface {
	FormatTools(tools []types.ToolDefinition) string
	FormatToolResult(result *types.ToolResult) string
	ParseToolCalls(response string) ([]ToolCall, error)
	ModelFamily() ModelFamily
}

type ToolCall struct {
	Name      string
	Arguments map[string]interface{}
	ID        string
}

type ToolCallParser func(response string) ([]ToolCall, error)

func ParseToolCallsByFamily(family ModelFamily) ToolCallParser {
	switch family {
	case ModelFamilyLlama3:
		return ParseLlama3ToolCalls
	case ModelFamilyQwen:
		return ParseQwenToolCalls
	case ModelFamilyMistral:
		return ParseMistralToolCalls
	case ModelFamilyClaude:
		return ParseClaudeToolCalls
	case ModelFamilyOpenAI:
		return ParseOpenAIToolCalls
	case ModelFamilyGLM4:
		return ParseGLM4ToolCalls
	default:
		return ParseClaudeToolCalls
	}
}

func ParseLlama3ToolCalls(response string) ([]ToolCall, error) {
	var calls []ToolCall
	re := regexp.MustCompile("<tool name=\"([^\"]+)\">(.*?)</tool>")
	matches := re.FindAllStringSubmatch(response, -1)
	for _, m := range matches {
		if len(m) >= 3 {
			calls = append(calls, ToolCall{Name: m[1]})
		}
	}
	return calls, nil
}

func ParseQwenToolCalls(response string) ([]ToolCall, error) {
	var calls []ToolCall
	re := regexp.MustCompile(`<invoke name="([^"]+)">`)
	matches := re.FindAllStringSubmatch(response, -1)
	for _, m := range matches {
		if len(m) >= 2 {
			calls = append(calls, ToolCall{Name: m[1]})
		}
	}
	return calls, nil
}

func ParseMistralToolCalls(response string) ([]ToolCall, error) {
	var calls []ToolCall
	re := regexp.MustCompile(`<tool>([^\s]+)</tool>`)
	matches := re.FindAllStringSubmatch(response, -1)
	for _, m := range matches {
		if len(m) >= 2 {
			calls = append(calls, ToolCall{Name: m[1]})
		}
	}
	return calls, nil
}

func ParseClaudeToolCalls(response string) ([]ToolCall, error) {
	var calls []ToolCall
	re := regexp.MustCompile(`<tool_call>(.*?)</tool_call>`)
	blockMatches := re.FindAllStringSubmatch(response, -1)
	for _, block := range blockMatches {
		if len(block) < 2 {
			continue
		}
		nameRe := regexp.MustCompile(`<tool name="([^\"]+)"`)
		nameMatch := nameRe.FindStringSubmatch(block[1])
		if len(nameMatch) >= 2 {
			calls = append(calls, ToolCall{Name: nameMatch[1]})
		}
	}
	return calls, nil
}

func ParseOpenAIToolCalls(response string) ([]ToolCall, error) {
	var calls []ToolCall
	var toolCalls []struct {
		Function struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"function"`
	}
	if err := json.Unmarshal([]byte(response), &toolCalls); err != nil {
		return nil, err
	}
	for _, tc := range toolCalls {
		args := make(map[string]interface{})
		json.Unmarshal(tc.Function.Arguments, &args)
		calls = append(calls, ToolCall{
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}
	return calls, nil
}

func ParseGLM4ToolCalls(response string) ([]ToolCall, error) {
	return ParseOpenAIToolCalls(response)
}

type llama3Formatter struct{}

func NewLlama3Formatter() ToolFormatter {
	return &llama3Formatter{}
}

func (f *llama3Formatter) FormatTools(tools []types.ToolDefinition) string {
	var sb strings.Builder
	sb.WriteString("Available tools:\n")
	for _, t := range tools {
		sb.WriteString("- ")
		sb.WriteString(t.Name)
		sb.WriteString(": ")
		sb.WriteString(t.Description)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (f *llama3Formatter) FormatToolResult(result *types.ToolResult) string {
	if result.Error != nil {
		return "Error: " + result.Error.Error()
	}
	return formatResultAsString(result.Content)
}

func (f *llama3Formatter) ParseToolCalls(response string) ([]ToolCall, error) {
	return ParseLlama3ToolCalls(response)
}

func (f *llama3Formatter) ModelFamily() ModelFamily {
	return ModelFamilyLlama3
}

type qwenFormatter struct{}

func NewQwenFormatter() ToolFormatter {
	return &qwenFormatter{}
}

func (f *qwenFormatter) FormatTools(tools []types.ToolDefinition) string {
	var sb strings.Builder
	sb.WriteString("## Tools\n\n")
	for _, t := range tools {
		sb.WriteString("### ")
		sb.WriteString(t.Name)
		sb.WriteString("\n\n")
		sb.WriteString(t.Description)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func (f *qwenFormatter) FormatToolResult(result *types.ToolResult) string {
	if result.Error != nil {
		return "Error: " + result.Error.Error()
	}
	return formatResultAsString(result.Content)
}

func (f *qwenFormatter) ParseToolCalls(response string) ([]ToolCall, error) {
	return ParseQwenToolCalls(response)
}

func (f *qwenFormatter) ModelFamily() ModelFamily {
	return ModelFamilyQwen
}

type mistralFormatter struct{}

func NewMistralFormatter() ToolFormatter {
	return &mistralFormatter{}
}

func (f *mistralFormatter) FormatTools(tools []types.ToolDefinition) string {
	var sb strings.Builder
	sb.WriteString("<tools>\n")
	for _, t := range tools {
		sb.WriteString("<tool name=\"")
		sb.WriteString(t.Name)
		sb.WriteString("\">")
		sb.WriteString(t.Description)
		sb.WriteString("</tool>\n")
	}
	sb.WriteString("</tools>")
	return sb.String()
}

func (f *mistralFormatter) FormatToolResult(result *types.ToolResult) string {
	if result.Error != nil {
		return "<error>" + result.Error.Error() + "</error>"
	}
	return "<result>" + formatResultAsString(result.Content) + "</result>"
}

func (f *mistralFormatter) ParseToolCalls(response string) ([]ToolCall, error) {
	return ParseMistralToolCalls(response)
}

func (f *mistralFormatter) ModelFamily() ModelFamily {
	return ModelFamilyMistral
}

type claudeFormatter struct{}

func NewClaudeFormatter() ToolFormatter {
	return &claudeFormatter{}
}

func (f *claudeFormatter) FormatTools(tools []types.ToolDefinition) string {
	var sb strings.Builder
	sb.WriteString("Tools:\n\n")
	for _, t := range tools {
		sb.WriteString("## ")
		sb.WriteString(t.Name)
		sb.WriteString("\n\n")
		sb.WriteString(t.Description)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func (f *claudeFormatter) FormatToolResult(result *types.ToolResult) string {
	if result.Error != nil {
		return "Error: " + result.Error.Error()
	}
	return formatResultAsString(result.Content)
}

func (f *claudeFormatter) ParseToolCalls(response string) ([]ToolCall, error) {
	return ParseClaudeToolCalls(response)
}

func (f *claudeFormatter) ModelFamily() ModelFamily {
	return ModelFamilyClaude
}

type openAIFormatter struct{}

func NewOpenAIFormatter() ToolFormatter {
	return &openAIFormatter{}
}

func (f *openAIFormatter) FormatTools(tools []types.ToolDefinition) string {
	toolsJSON, _ := json.Marshal(tools)
	return string(toolsJSON)
}

func (f *openAIFormatter) FormatToolResult(result *types.ToolResult) string {
	if result.Error != nil {
		return result.Error.Error()
	}
	b, _ := json.Marshal(result.Content)
	return string(b)
}

func (f *openAIFormatter) ParseToolCalls(response string) ([]ToolCall, error) {
	return ParseOpenAIToolCalls(response)
}

func (f *openAIFormatter) ModelFamily() ModelFamily {
	return ModelFamilyOpenAI
}

type glm4Formatter struct{}

func NewGLM4Formatter() ToolFormatter {
	return &glm4Formatter{}
}

func (f *glm4Formatter) FormatTools(tools []types.ToolDefinition) string {
	toolsJSON, _ := json.Marshal(tools)
	return string(toolsJSON)
}

func (f *glm4Formatter) FormatToolResult(result *types.ToolResult) string {
	if result.Error != nil {
		return result.Error.Error()
	}
	b, _ := json.Marshal(result.Content)
	return string(b)
}

func (f *glm4Formatter) ParseToolCalls(response string) ([]ToolCall, error) {
	return ParseGLM4ToolCalls(response)
}

func (f *glm4Formatter) ModelFamily() ModelFamily {
	return ModelFamilyGLM4
}

func NewFormatter(family ModelFamily) ToolFormatter {
	switch family {
	case ModelFamilyLlama3:
		return NewLlama3Formatter()
	case ModelFamilyQwen:
		return NewQwenFormatter()
	case ModelFamilyMistral:
		return NewMistralFormatter()
	case ModelFamilyClaude:
		return NewClaudeFormatter()
	case ModelFamilyOpenAI:
		return NewOpenAIFormatter()
	case ModelFamilyGLM4:
		return NewGLM4Formatter()
	default:
		return NewClaudeFormatter()
	}
}

func formatResultAsString(content interface{}) string {
	switch c := content.(type) {
	case string:
		return c
	case []byte:
		return string(c)
	default:
		b, _ := json.Marshal(c)
		return string(b)
	}
}
