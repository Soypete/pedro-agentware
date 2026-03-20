package format

import (
	"context"
	"strings"

	"github.com/soypete/pedro-agentware/types"
)

type MCPBridge interface {
	Connect(ctx context.Context, serverName string) error
	CallTool(ctx context.Context, server string, tool string, args map[string]interface{}) (*types.ToolResult, error)
	ListTools(server string) []types.ToolDefinition
}

type HybridBridge struct {
	directBridge *DirectBridge
	mcpBridge    MCPBridge
	useMCP       bool
}

func NewHybridBridge(executor types.ToolExecutor, mcpBridge MCPBridge, modelFamily ModelFamily) *HybridBridge {
	useMCP := IsMCPCapable(modelFamily)

	return &HybridBridge{
		directBridge: NewDirectBridge(executor, modelFamily),
		mcpBridge:    mcpBridge,
		useMCP:       useMCP,
	}
}

func (h *HybridBridge) FormatTools(tools []types.ToolDefinition) string {
	if h.useMCP {
		return h.formatToolsForMCP(tools)
	}
	return h.directBridge.FormatTools(tools)
}

func (h *HybridBridge) formatToolsForMCP(tools []types.ToolDefinition) string {
	var sb strings.Builder
	sb.WriteString("Available MCP tools:\n")
	for _, t := range tools {
		sb.WriteString("- ")
		sb.WriteString(t.Name)
		sb.WriteString(": ")
		sb.WriteString(t.Description)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (h *HybridBridge) FormatToolResult(result *types.ToolResult) string {
	return h.directBridge.FormatToolResult(result)
}

func (h *HybridBridge) ParseToolCalls(response string) ([]ToolCall, error) {
	return h.directBridge.ParseToolCalls(response)
}

func (h *HybridBridge) ExecuteTool(ctx context.Context, call ToolCall) (*types.ToolResult, error) {
	if h.useMCP && h.mcpBridge != nil {
		return h.executeViaMCP(ctx, call)
	}
	return h.directBridge.ExecuteTool(ctx, call)
}

func (h *HybridBridge) executeViaMCP(ctx context.Context, call ToolCall) (*types.ToolResult, error) {
	if !strings.Contains(call.Name, "/") {
		return &types.ToolResult{
			Error: ErrInvalidToolFormat,
		}, ErrInvalidToolFormat
	}

	parts := strings.SplitN(call.Name, "/", 2)
	if len(parts) != 2 {
		return &types.ToolResult{
			Error: ErrInvalidToolFormat,
		}, ErrInvalidToolFormat
	}

	server, toolName := parts[0], parts[1]
	return h.mcpBridge.CallTool(ctx, server, toolName, call.Arguments)
}

func (h *HybridBridge) ListTools() []types.ToolDefinition {
	if h.useMCP && h.mcpBridge != nil {
		return h.listMCPTools()
	}
	return h.directBridge.ListTools()
}

func (h *HybridBridge) listMCPTools() []types.ToolDefinition {
	return nil
}

func (h *HybridBridge) SetUseMCP(useMCP bool) {
	h.useMCP = useMCP
}

func (h *HybridBridge) UseMCP() bool {
	return h.useMCP
}

func (h *HybridBridge) ModelFamily() ModelFamily {
	return h.directBridge.ModelFamily()
}

func (h *HybridBridge) SwitchToDirect() {
	h.useMCP = false
}

func (h *HybridBridge) SwitchToMCP() {
	h.useMCP = true
}

func splitToolName(fullName string) []string {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) == 2 {
		return parts
	}
	return []string{"default", fullName}
}

var ErrInvalidToolFormat = &InvalidToolFormatError{}

type InvalidToolFormatError struct{}

func (e *InvalidToolFormatError) Error() string {
	return "tool name must be in format 'server/tool' for MCP calls"
}
