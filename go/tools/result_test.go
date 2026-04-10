package tools

import (
	"testing"
)

func TestResult(t *testing.T) {
	result := Result{
		Success:       true,
		Output:        "test output",
		Error:         "",
		ModifiedFiles: []string{"file1.go", "file2.go"},
		Metadata:      map[string]any{"duration": 100},
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.Output != "test output" {
		t.Errorf("expected Output 'test output', got '%s'", result.Output)
	}
	if len(result.ModifiedFiles) != 2 {
		t.Errorf("expected 2 ModifiedFiles, got %d", len(result.ModifiedFiles))
	}
	if result.Metadata["duration"].(int) != 100 {
		t.Errorf("expected duration 100, got %v", result.Metadata["duration"])
	}
}

func TestResultError(t *testing.T) {
	result := Result{
		Success: false,
		Error:   "something went wrong",
	}

	if result.Success {
		t.Error("expected Success to be false")
	}
	if result.Error != "something went wrong" {
		t.Errorf("expected Error 'something went wrong', got '%s'", result.Error)
	}
}
