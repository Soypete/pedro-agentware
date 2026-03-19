package middleware

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPolicyFromString(t *testing.T) {
	yaml := `
rules:
  - name: allow-all
    tools:
      - "*"
    action: allow
default_deny: false
`
	policy, err := LoadPolicyFromString(yaml)
	if err != nil {
		t.Fatalf("LoadPolicyFromString failed: %v", err)
	}
	if len(policy.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(policy.Rules))
	}
	if policy.Rules[0].Name != "allow-all" {
		t.Errorf("expected rule name 'allow-all', got %s", policy.Rules[0].Name)
	}
	if policy.DefaultDeny {
		t.Error("expected DefaultDeny to be false")
	}
}

func TestLoadPolicyFromBytes(t *testing.T) {
	yaml := []byte(`
rules:
  - name: deny-tool
    tools:
      - "dangerous"
    action: deny
default_deny: true
`)
	policy, err := LoadPolicyFromBytes(yaml)
	if err != nil {
		t.Fatalf("LoadPolicyFromBytes failed: %v", err)
	}
	if len(policy.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(policy.Rules))
	}
	if policy.Rules[0].Action != ActionDeny {
		t.Errorf("expected ActionDeny, got %v", policy.Rules[0].Action)
	}
	if !policy.DefaultDeny {
		t.Error("expected DefaultDeny to be true")
	}
}

func TestLoadPolicyFromBytes_Invalid(t *testing.T) {
	_, err := LoadPolicyFromBytes([]byte("invalid: yaml: [[["))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "policy.yaml")
	yaml := `
rules:
  - name: test-rule
    tools:
      - "foo"
      - "bar"
    action: allow
default_deny: true
`
	if err := os.WriteFile(policyPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	policy, err := LoadPolicy(policyPath)
	if err != nil {
		t.Fatalf("LoadPolicy failed: %v", err)
	}
	if len(policy.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(policy.Rules))
	}
}

func TestLoadPolicy_FileNotFound(t *testing.T) {
	_, err := LoadPolicy("/nonexistent/path/policy.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadPolicy_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "policy.yaml")
	invalidYAML := "invalid: yaml: content: ["
	if err := os.WriteFile(policyPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadPolicy(policyPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
