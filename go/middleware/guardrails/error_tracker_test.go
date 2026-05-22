package guardrails

import (
	"errors"
	"testing"
	"time"
)

func TestErrorTracker_RecordError(t *testing.T) {
	et := NewErrorTracker()
	et.RecordError("session1", "tool1", map[string]interface{}{"key": "value"}, errors.New("test error"), ErrCategoryUnknown)

	count := et.GetErrorCount("session1", "tool1")
	if count != 1 {
		t.Errorf("expected 1 error, got %d", count)
	}
}

func TestErrorTracker_GetErrorCount(t *testing.T) {
	et := NewErrorTracker()
	et.RecordError("session1", "tool1", nil, errors.New("error1"), ErrCategoryTimeout)
	et.RecordError("session1", "tool1", nil, errors.New("error2"), ErrCategoryTimeout)
	et.RecordError("session1", "tool2", nil, errors.New("error3"), ErrCategoryNotFound)

	if et.GetErrorCount("session1", "tool1") != 2 {
		t.Error("expected 2 errors for tool1")
	}
	if et.GetErrorCount("session1", "tool2") != 1 {
		t.Error("expected 1 error for tool2")
	}
}

func TestErrorTracker_GetRecentErrors(t *testing.T) {
	et := NewErrorTracker()
	et.RecordError("session1", "tool1", nil, errors.New("error1"), ErrCategoryUnknown)

	errors := et.GetRecentErrors("session1")
	if len(errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(errors))
	}
}

func TestErrorTracker_GetErrorsByCategory(t *testing.T) {
	et := NewErrorTracker()
	et.RecordError("session1", "tool1", nil, errors.New("timeout"), ErrCategoryTimeout)
	et.RecordError("session1", "tool1", nil, errors.New("not found"), ErrCategoryNotFound)

	timeoutErrors := et.GetErrorsByCategory("session1", ErrCategoryTimeout)
	if len(timeoutErrors) != 1 {
		t.Error("expected 1 timeout error")
	}
}

func TestErrorTracker_IsErrorRateExceeded(t *testing.T) {
	et := NewErrorTracker()
	et.maxErrorsPerTool = 3

	for i := 0; i < 3; i++ {
		et.RecordError("session1", "tool1", nil, errors.New("error"), ErrCategoryUnknown)
	}

	if !et.IsErrorRateExceeded("session1", "tool1") {
		t.Error("expected error rate exceeded")
	}
}

func TestErrorTracker_ResetSession(t *testing.T) {
	et := NewErrorTracker()
	et.RecordError("session1", "tool1", nil, errors.New("error"), ErrCategoryUnknown)

	et.ResetSession("session1")
	count := et.GetErrorCount("session1", "tool1")
	if count != 0 {
		t.Error("expected 0 errors after reset")
	}
}

func TestErrorTracker_ShouldBlockTool(t *testing.T) {
	et := NewErrorTracker()
	et.maxErrorsPerTool = 2

	et.RecordError("session1", "tool1", nil, errors.New("error1"), ErrCategoryUnknown)
	et.RecordError("session1", "tool1", nil, errors.New("error2"), ErrCategoryUnknown)

	if !et.ShouldBlockTool("session1", "tool1") {
		t.Error("expected tool to be blocked")
	}
	if et.ShouldBlockTool("session1", "tool2") {
		t.Error("expected tool2 not to be blocked")
	}
}

func TestErrorTracker_SetThresholds(t *testing.T) {
	et := NewErrorTracker()
	et.SetThresholds(10, time.Minute*10)

	if et.maxErrorsPerTool != 10 {
		t.Errorf("expected maxErrorsPerTool 10, got %d", et.maxErrorsPerTool)
	}
	if et.windowDuration != time.Minute*10 {
		t.Error("expected windowDuration 10 minutes")
	}
}
