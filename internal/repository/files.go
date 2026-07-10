package repository

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"biztop/internal/domain"
)

// LoadRules reads rules.yml, the single source of truth for business rules.
func LoadRules(path string) (domain.Rules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Rules{}, fmt.Errorf("cannot read rules file: %w", err)
	}
	var rules domain.Rules
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return domain.Rules{}, fmt.Errorf("invalid %s: %w", path, err)
	}
	return rules, nil
}

// ReadObjectivesDoc returns the raw markdown brief, "" when absent.
func ReadObjectivesDoc(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// LoadEstimate returns the cached Attio estimate, nil when absent.
func LoadEstimate(path string) json.RawMessage {
	data, err := os.ReadFile(path)
	if err != nil || !json.Valid(data) {
		return nil
	}
	return data
}

func SaveEstimate(path string, estimate map[string]any) (json.RawMessage, error) {
	data, err := json.MarshalIndent(estimate, "", "  ")
	if err != nil {
		return nil, err
	}
	return data, os.WriteFile(path, append(data, '\n'), 0o644)
}
