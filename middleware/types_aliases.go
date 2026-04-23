package middleware

import (
	"github.com/soypete/pedro-agentware/middleware/types"

	"github.com/soypete/pedro-agentware/moderation"
)

type Action = types.Action

type ModerationDecision = moderation.ModerationDecision
type TimeoutUserParams = moderation.TimeoutUserParams
type BanUserParams = moderation.BanUserParams
type UnbanUserParams = moderation.UnbanUserParams
type ModeratorParams = moderation.ModeratorParams
type VIPParams = moderation.VIPParams
type DeleteMessageParams = moderation.DeleteMessageParams
type ChatModeParams = moderation.ChatModeParams
type PollParams = moderation.PollParams
type EndPollParams = moderation.EndPollParams
type PredictionParams = moderation.PredictionParams
type ResolvePredictionParams = moderation.ResolvePredictionParams
type CancelPredictionParams = moderation.CancelPredictionParams
type AnnouncementParams = moderation.AnnouncementParams
type ShoutoutParams = moderation.ShoutoutParams
type NoActionParams = moderation.NoActionParams
type WarnUserParams = moderation.WarnUserParams

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
