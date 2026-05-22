package guardrails

import "time"

type ErrorCategory string

const (
	ErrCategoryTimeout     ErrorCategory = "timeout"
	ErrCategoryNotFound    ErrorCategory = "not_found"
	ErrCategoryInvalidArgs ErrorCategory = "invalid_args"
	ErrCategoryPermission  ErrorCategory = "permission"
	ErrCategoryRateLimit   ErrorCategory = "rate_limit"
	ErrCategoryUnknown     ErrorCategory = "unknown"
)

type ToolError struct {
	Timestamp  time.Time
	Tool       string
	Args       map[string]interface{}
	Category   ErrorCategory
	Message    string
	SessionID  string
	RetryCount int
}

type ErrorTracker struct {
	errors           map[string][]ToolError
	maxErrorsPerTool int
	windowDuration   time.Duration
}

func NewErrorTracker() *ErrorTracker {
	return &ErrorTracker{
		errors:           make(map[string][]ToolError),
		maxErrorsPerTool: 5,
		windowDuration:   time.Minute * 5,
	}
}

func (et *ErrorTracker) SetThresholds(maxErrors int, window time.Duration) {
	et.maxErrorsPerTool = maxErrors
	et.windowDuration = window
}

func (et *ErrorTracker) RecordError(sessionID, tool string, args map[string]interface{}, err error, category ErrorCategory) {
	if et.errors[sessionID] == nil {
		et.errors[sessionID] = make([]ToolError, 0)
	}

	toolErr := ToolError{
		Timestamp:  time.Now(),
		Tool:       tool,
		Args:       args,
		Category:   category,
		Message:    err.Error(),
		SessionID:  sessionID,
		RetryCount: et.getRetryCount(sessionID, tool),
	}

	et.errors[sessionID] = append(et.errors[sessionID], toolErr)
	et.pruneOldErrors(sessionID)
}

func (et *ErrorTracker) getRetryCount(sessionID, tool string) int {
	count := 0
	for _, err := range et.errors[sessionID] {
		if err.Tool == tool {
			count++
		}
	}
	return count
}

func (et *ErrorTracker) pruneOldErrors(sessionID string) {
	if len(et.errors[sessionID]) <= et.maxErrorsPerTool {
		return
	}

	cutoff := time.Now().Add(-et.windowDuration)
	var filtered []ToolError
	for _, err := range et.errors[sessionID] {
		if err.Timestamp.After(cutoff) {
			filtered = append(filtered, err)
		}
	}
	et.errors[sessionID] = filtered
}

func (et *ErrorTracker) GetErrorCount(sessionID, tool string) int {
	count := 0
	for _, err := range et.errors[sessionID] {
		if err.Tool == tool {
			count++
		}
	}
	return count
}

func (et *ErrorTracker) GetRecentErrors(sessionID string) []ToolError {
	et.pruneOldErrors(sessionID)
	result := make([]ToolError, len(et.errors[sessionID]))
	copy(result, et.errors[sessionID])
	return result
}

func (et *ErrorTracker) GetErrorsByCategory(sessionID string, category ErrorCategory) []ToolError {
	var result []ToolError
	for _, err := range et.errors[sessionID] {
		if err.Category == category {
			result = append(result, err)
		}
	}
	return result
}

func (et *ErrorTracker) IsErrorRateExceeded(sessionID, tool string) bool {
	count := et.GetErrorCount(sessionID, tool)
	return count >= et.maxErrorsPerTool
}

func (et *ErrorTracker) ResetSession(sessionID string) {
	delete(et.errors, sessionID)
}

func (et *ErrorTracker) ShouldBlockTool(sessionID, tool string) bool {
	return et.IsErrorRateExceeded(sessionID, tool)
}
