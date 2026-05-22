package llm

import (
	"testing"
)

func TestContextWindowManager_UpdateTokenCount(t *testing.T) {
	mgr := NewContextWindowManager(1000, nil)

	mgr.UpdateTokenCount(500)

	tokens, atThreshold := mgr.Check([]Message{
		{Role: RoleUser, Content: "test"},
	})
	if tokens != 500 {
		t.Errorf("expected 500 tokens, got %d", tokens)
	}
	if atThreshold {
		t.Error("should not be at threshold with 500 tokens")
	}
}

func TestContextWindowManager_Check_UsesActualCount(t *testing.T) {
	counter := func(messages []Message) int {
		return 800
	}
	mgr := NewContextWindowManager(1000, counter)
	mgr.SetCompactionRatio(0.75)

	tokens, atThreshold := mgr.Check([]Message{{Role: RoleUser, Content: "test"}})
	if tokens != 800 {
		t.Errorf("expected 800 tokens from counter, got %d", tokens)
	}
	if !atThreshold {
		t.Error("should be at threshold with 800 tokens (75% of 1000)")
	}

	mgr.UpdateTokenCount(600)

	tokens, atThreshold = mgr.Check([]Message{{Role: RoleUser, Content: "test"}})
	if tokens != 600 {
		t.Errorf("expected 600 tokens from UpdateTokenCount, got %d", tokens)
	}
	if atThreshold {
		t.Error("should not be at threshold with 600 tokens (60% of 1000)")
	}
}

func TestContextWindowManager_ShouldCompact_UsesActualCount(t *testing.T) {
	counter := func(messages []Message) int {
		return 900
	}
	mgr := NewContextWindowManager(1000, counter)
	mgr.SetCompactionRatio(0.75)

	if !mgr.ShouldCompact([]Message{{Role: RoleUser, Content: "test"}}) {
		t.Error("should compact with 900 tokens (90% of 1000)")
	}

	mgr.UpdateTokenCount(700)

	if mgr.ShouldCompact([]Message{{Role: RoleUser, Content: "test"}}) {
		t.Error("should not compact with 700 tokens (70% of 1000)")
	}
}

func TestContextWindowManager_Compact_ResetsTokenCount(t *testing.T) {
	mgr := NewContextWindowManager(1000, nil)
	mgr.SetCompactionRatio(0.75)

	mgr.UpdateTokenCount(1500)

	compacted, err := mgr.Compact([]Message{
		{Role: RoleUser, Content: "short"},
	})
	if err != nil {
		t.Fatalf("Compact failed: %v", err)
	}

	tokens, _ := mgr.Check(compacted)
	if tokens == 1500 {
		t.Error("token count should be reset after compaction, got same value")
	}
	if tokens >= 100 {
		t.Errorf("expected low token count after compaction, got %d", tokens)
	}
}

func TestContextWindowManager_ThreadSafety(t *testing.T) {
	mgr := NewContextWindowManager(1000, nil)

	done := make(chan bool)
	go func() {
		for i := 0; i < 1000; i++ {
			mgr.UpdateTokenCount(i)
		}
		done <- true
	}()

	for i := 0; i < 1000; i++ {
		mgr.Check([]Message{{Role: RoleUser, Content: "test"}})
	}

	<-done
}

func TestDefaultCounter(t *testing.T) {
	messages := []Message{
		{Role: RoleUser, Content: "Hello world test content"},
		{Role: RoleAssistant, Content: "Response with some text"},
	}

	count := DefaultCounter(messages)

	expected := (len("Hello world test content") / 4) + len("user") + 4
	expected += (len("Response with some text") / 4) + len("assistant") + 4

	if count != expected {
		t.Errorf("expected %d, got %d", expected, count)
	}
}
