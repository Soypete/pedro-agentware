package moderation

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ModAction struct {
	ID                    uuid.UUID       `db:"id"`
	CreatedAt             time.Time       `db:"created_at"`
	TriggerMessageID      string          `db:"trigger_message_id"`
	TriggerUsername       string          `db:"trigger_username"`
	TriggerMessageContent string          `db:"trigger_message_content"`
	LLMModel              string          `db:"llm_model"`
	LLMReasoning          string          `db:"llm_reasoning"`
	ToolCallName          string          `db:"tool_call_name"`
	ToolCallParams        json.RawMessage `db:"tool_call_params"`
	TargetUsername        string          `db:"target_username"`
	TargetUserID          string          `db:"target_user_id"`
	TwitchAPIResponse     json.RawMessage `db:"twitch_api_response"`
	Success               bool            `db:"success"`
	ErrorMessage          string          `db:"error_message"`
	ChannelID             string          `db:"channel_id"`
	ChannelName           string          `db:"channel_name"`
}

type ModerationDecision struct {
	ShouldAct    bool
	ToolCall     string
	ToolParams   map[string]interface{}
	Reasoning    string
	TargetUserID string
}

type TimeoutUserParams struct {
	Username        string `json:"username"`
	DurationSeconds int    `json:"duration_seconds"`
	Reason          string `json:"reason"`
}

type BanUserParams struct {
	Username string `json:"username"`
	Reason   string `json:"reason"`
}

type UnbanUserParams struct {
	Username string `json:"username"`
}

type ModeratorParams struct {
	Username string `json:"username"`
}

type VIPParams struct {
	Username string `json:"username"`
}

type DeleteMessageParams struct {
	MessageID string `json:"message_id"`
}

type ChatModeParams struct {
	Enabled         bool `json:"enabled"`
	DurationMinutes int  `json:"duration_minutes,omitempty"`
	DelaySeconds    int  `json:"delay_seconds,omitempty"`
}

type PollParams struct {
	Title           string   `json:"title"`
	Choices         []string `json:"choices"`
	DurationSeconds int      `json:"duration_seconds"`
}

type EndPollParams struct {
	PollID string `json:"poll_id"`
	Status string `json:"status"`
}

type PredictionParams struct {
	Title           string   `json:"title"`
	Outcomes        []string `json:"outcomes"`
	DurationSeconds int      `json:"duration_seconds"`
}

type ResolvePredictionParams struct {
	PredictionID     string `json:"prediction_id"`
	WinningOutcomeID string `json:"winning_outcome_id"`
}

type CancelPredictionParams struct {
	PredictionID string `json:"prediction_id"`
}

type AnnouncementParams struct {
	Message string `json:"message"`
	Color   string `json:"color"`
}

type ShoutoutParams struct {
	Username string `json:"username"`
}

type NoActionParams struct {
	Reason string `json:"reason"`
}

type WarnUserParams struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}
