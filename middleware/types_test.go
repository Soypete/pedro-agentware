package middleware

import (
	"testing"
	"time"
)

func TestDecision_IsAllowed(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		expected bool
	}{
		{"allow action returns true", ActionAllow, true},
		{"deny action returns false", ActionDeny, false},
		{"filter action returns false", ActionFilter, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Decision{Action: tt.action}
			if got := d.IsAllowed(); got != tt.expected {
				t.Errorf("IsAllowed() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDecision_IsDenied(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		expected bool
	}{
		{"allow action returns false", ActionAllow, false},
		{"deny action returns true", ActionDeny, true},
		{"filter action returns false", ActionFilter, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Decision{Action: tt.action}
			if got := d.IsDenied(); got != tt.expected {
				t.Errorf("IsDenied() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDecision_IsFiltered(t *testing.T) {
	tests := []struct {
		name     string
		action   Action
		expected bool
	}{
		{"allow action returns false", ActionAllow, false},
		{"deny action returns false", ActionDeny, false},
		{"filter action returns true", ActionFilter, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Decision{Action: tt.action}
			if got := d.IsFiltered(); got != tt.expected {
				t.Errorf("IsFiltered() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCallerContext(t *testing.T) {
	ctx := CallerContext{
		Trusted:   true,
		Role:      "admin",
		UserID:    "user123",
		SessionID: "session456",
		Source:    "cli",
		Metadata:  map[string]interface{}{"key": "value"},
	}

	if ctx.Trusted != true {
		t.Error("Trusted should be true")
	}
	if ctx.Role != "admin" {
		t.Error("Role should be admin")
	}
}

func TestToolResult(t *testing.T) {
	result := ToolResult{
		Content:  "test content",
		Error:    nil,
		IsStream: false,
		Metadata: map[string]interface{}{"foo": "bar"},
	}

	if result.Content != "test content" {
		t.Error("Content mismatch")
	}
	if result.Error != nil {
		t.Error("Error should be nil")
	}
}

func TestToolDefinition(t *testing.T) {
	def := ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{"type": "object"},
	}

	if def.Name != "test_tool" {
		t.Error("Name mismatch")
	}
}

func TestDecision_Timestamp(t *testing.T) {
	now := time.Now()
	d := Decision{
		Timestamp: now,
		Tool:      "test",
		Action:    ActionAllow,
	}

	if !d.Timestamp.Equal(now) {
		t.Error("Timestamp mismatch")
	}
}
