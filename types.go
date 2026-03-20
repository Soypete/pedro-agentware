package middleware

import (
	"github.com/soypete/pedro-agentware/types"
)

type Action = types.Action

const (
	ActionAllow  = types.ActionAllow
	ActionDeny   = types.ActionDeny
	ActionFilter = types.ActionFilter
)

type ToolResult = types.ToolResult
type ToolDefinition = types.ToolDefinition
type CallerContext = types.CallerContext
type Decision = types.Decision
type ToolExecutor = types.ToolExecutor
type RateLimit = types.RateLimit
type Schema = types.Schema
type Condition = types.Condition
type QuickFilter = types.QuickFilter
type Rule = types.Rule
type Policy = types.Policy
