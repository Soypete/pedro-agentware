package llmcontext

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCompactor_DefaultConfig(t *testing.T) {
	config := DefaultCompactionConfig()
	if config.Threshold != 0.75 {
		t.Errorf("Threshold = %v, want 0.75", config.Threshold)
	}
	if config.KeepRecent != 3 {
		t.Errorf("KeepRecent = %v, want 3", config.KeepRecent)
	}
	if config.ContextLimit != 100000 {
		t.Errorf("ContextLimit = %v, want 100000", config.ContextLimit)
	}
}

func TestCompactor_NewCompactor(t *testing.T) {
	tmpDir := t.TempDir()

	config := CompactionConfig{
		Threshold:    0.5,
		KeepRecent:   2,
		ContextLimit: 50000,
	}

	compactor := NewCompactor(tmpDir, config)
	if compactor == nil {
		t.Fatal("NewCompactor returned nil")
	}
	if compactor.config.Threshold != 0.5 {
		t.Errorf("config.Threshold = %v, want 0.5", compactor.config.Threshold)
	}
	if compactor.config.KeepRecent != 2 {
		t.Errorf("config.KeepRecent = %v, want 2", compactor.config.KeepRecent)
	}
}

func TestCompactor_NewCompactor_DefaultOnZero(t *testing.T) {
	tmpDir := t.TempDir()

	config := CompactionConfig{}
	compactor := NewCompactor(tmpDir, config)

	if compactor.config.Threshold != 0.75 {
		t.Errorf("Threshold = %v, want 0.75", compactor.config.Threshold)
	}
}

func TestCompactor_ShouldCompact_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	compactor := NewCompactor(tmpDir, DefaultCompactionConfig())

	if compactor.ShouldCompact() {
		t.Error("ShouldCompact should be false with no files")
	}
}

func TestCompactor_ShouldCompact_UnderThreshold(t *testing.T) {
	tmpDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(tmpDir, "001-prompt.txt"), []byte("test prompt"), 0644)

	compactor := NewCompactor(tmpDir, CompactionConfig{
		Threshold:    0.75,
		KeepRecent:   3,
		ContextLimit: 100000,
	})

	if compactor.ShouldCompact() {
		t.Error("ShouldCompact should be false when under threshold")
	}
}

func TestCompactor_ShouldCompact_OverThreshold(t *testing.T) {
	tmpDir := t.TempDir()

	for i := 1; i <= 10; i++ {
		_ = os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("%03d-prompt.txt", i)), []byte("test prompt with lots of content to exceed token threshold"), 0644)
		_ = os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("%03d-response.txt", i)), []byte("test response with lots of content to exceed token threshold"), 0644)
	}

	compactor := NewCompactor(tmpDir, CompactionConfig{
		Threshold:    0.1,
		KeepRecent:   3,
		ContextLimit: 1000,
	})

	if !compactor.ShouldCompact() {
		t.Error("ShouldCompact should be true when over threshold")
	}
}

func TestCompactor_Compact_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	compactor := NewCompactor(tmpDir, DefaultCompactionConfig())

	result, err := compactor.Compact()
	if err != nil {
		t.Errorf("Compact failed: %v", err)
	}
	if result != "" {
		t.Errorf("Compact returned non-empty for empty dir: %q", result)
	}
}

func TestCompactor_Compact_UnderKeepRecent(t *testing.T) {
	tmpDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(tmpDir, "001-prompt.txt"), []byte("test prompt"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "002-prompt.txt"), []byte("test prompt 2"), 0644)

	compactor := NewCompactor(tmpDir, CompactionConfig{
		KeepRecent: 3,
	})

	result, err := compactor.Compact()
	if err != nil {
		t.Errorf("Compact failed: %v", err)
	}
	if result != "" {
		t.Errorf("Compact returned non-empty when under KeepRecent: %q", result)
	}
}

func TestCompactor_Compact_OverKeepRecent(t *testing.T) {
	tmpDir := t.TempDir()

	for i := 1; i <= 5; i++ {
		_ = os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("%03d-prompt.txt", i)), []byte("test prompt"), 0644)
		_ = os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("%03d-tool-calls.json", i)), []byte(`[{"Name": "bash", "Args": {"command": "ls"}}]`), 0644)
	}

	compactor := NewCompactor(tmpDir, CompactionConfig{
		KeepRecent: 2,
	})

	result, err := compactor.Compact()
	if err != nil {
		t.Errorf("Compact failed: %v", err)
	}
	if result == "" {
		t.Error("Compact returned empty when should have summary")
	}
}

func TestCompactor_GetStats(t *testing.T) {
	tmpDir := t.TempDir()

	for i := 1; i <= 5; i++ {
		_ = os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("%03d-prompt.txt", i)), []byte("test"), 0644)
	}

	compactor := NewCompactor(tmpDir, DefaultCompactionConfig())

	stats := compactor.GetStats()
	if stats.TotalRounds != 5 {
		t.Errorf("TotalRounds = %d, want 5", stats.TotalRounds)
	}
	if stats.RecentRounds != 3 {
		t.Errorf("RecentRounds = %d, want 3", stats.RecentRounds)
	}
}

func TestCompactor_EstimateTokenCount(t *testing.T) {
	tmpDir := t.TempDir()

	content := "this is a test prompt with exactly twenty four characters"
	_ = os.WriteFile(filepath.Join(tmpDir, "001-prompt.txt"), []byte(content), 0644)

	compactor := NewCompactor(tmpDir, DefaultCompactionConfig())

	estimated := compactor.estimateTokenCount()
	if estimated == 0 {
		t.Error("estimateTokenCount returned 0")
	}
}

func TestCompactor_getPromptFiles(t *testing.T) {
	tmpDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(tmpDir, "001-prompt.txt"), []byte("test"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "002-prompt.txt"), []byte("test"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "003-other.txt"), []byte("test"), 0644)

	compactor := NewCompactor(tmpDir, DefaultCompactionConfig())

	files, err := compactor.getPromptFiles()
	if err != nil {
		t.Errorf("getPromptFiles failed: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("len(files) = %d, want 2", len(files))
	}
}
