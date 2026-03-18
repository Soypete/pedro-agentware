package middleware

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadPolicy(path string) (Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("failed to read policy file: %w", err)
	}

	var policy Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return Policy{}, fmt.Errorf("failed to parse policy file: %w", err)
	}

	return policy, nil
}

func LoadPolicyFromBytes(data []byte) (Policy, error) {
	var policy Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return Policy{}, fmt.Errorf("failed to parse policy: %w", err)
	}
	return policy, nil
}

func LoadPolicyFromString(data string) (Policy, error) {
	return LoadPolicyFromBytes([]byte(data))
}
