package format

import (
	"context"
	"strings"

	"github.com/pedro/agent-middleware/types"
)

type DirectBridge struct {
	executor   types.ToolExecutor
	formatter  ToolFormatter
	toolFilter ToolFilter
}

type ToolFilter func(tools []types.ToolDefinition) []types.ToolDefinition

func NewDirectBridge(executor types.ToolExecutor, family ModelFamily) *DirectBridge {
	return &DirectBridge{
		executor:   executor,
		formatter:  NewFormatter(family),
		toolFilter: nil,
	}
}

func (b *DirectBridge) FormatTools(tools []types.ToolDefinition) string {
	filtered := tools
	if b.toolFilter != nil {
		filtered = b.toolFilter(tools)
	}
	return b.formatter.FormatTools(filtered)
}

func (b *DirectBridge) FormatToolResult(result *types.ToolResult) string {
	return b.formatter.FormatToolResult(result)
}

func (b *DirectBridge) ParseToolCalls(response string) ([]ToolCall, error) {
	return b.formatter.ParseToolCalls(response)
}

func (b *DirectBridge) ExecuteTool(ctx context.Context, call ToolCall) (*types.ToolResult, error) {
	return b.executor.CallTool(ctx, call.Name, call.Arguments)
}

func (b *DirectBridge) ExecuteToolByName(ctx context.Context, name string, args map[string]interface{}) (*types.ToolResult, error) {
	return b.executor.CallTool(ctx, name, args)
}

func (b *DirectBridge) ListTools() []types.ToolDefinition {
	tools := b.executor.ListTools()
	if b.toolFilter != nil {
		return b.toolFilter(tools)
	}
	return tools
}

func (b *DirectBridge) SetToolFilter(filter ToolFilter) {
	b.toolFilter = filter
}

func (b *DirectBridge) ModelFamily() ModelFamily {
	return b.formatter.ModelFamily()
}

func IsMCPCapable(family ModelFamily) bool {
	switch family {
	case ModelFamilyClaude, ModelFamilyOpenAI, ModelFamilyGLM4:
		return true
	default:
		return false
	}
}

func NewDirectBridgeForNonMCP(executor types.ToolExecutor, modelResponse string) (*DirectBridge, error) {
	family := DetectModelFamily(modelResponse)
	if IsMCPCapable(family) {
		return nil, ErrMCPRequired
	}
	bridge := NewDirectBridge(executor, family)
	return bridge, nil
}

func DetectModelFamily(response string) ModelFamily {
	lowerResp := strings.ToLower(response)

	if strings.Contains(lowerResp, "anthropic") || strings.Contains(lowerResp, "claude") {
		return ModelFamilyClaude
	}
	if strings.Contains(lowerResp, "openai") || strings.Contains(lowerResp, "gpt-") {
		return ModelFamilyOpenAI
	}
	if strings.Contains(lowerResp, "glm-") || strings.Contains(lowerResp, "glm4") {
		return ModelFamilyGLM4
	}
	if strings.Contains(lowerResp, "llama") || strings.Contains(lowerResp, "meta") {
		return ModelFamilyLlama3
	}
	if strings.Contains(lowerResp, "qwen") || strings.Contains(lowerResp, "alibaba") {
		return ModelFamilyQwen
	}
	if strings.Contains(lowerResp, "mistral") {
		return ModelFamilyMistral
	}

	return ModelFamilyClaude
}

var ErrMCPRequired = &MCPCapabilityError{}

type MCPCapabilityError struct{}

func (e *MCPCapabilityError) Error() string {
	return "MCP-capable model required for this operation"
}
