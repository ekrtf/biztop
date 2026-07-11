package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"biztop/internal/domain"
)

// LoadRules reads rules.yml, the single source of truth for business rules.
// Decoding is strict: an unknown key (e.g. a typo like "exclude" instead of
// "exclude_patterns") is an error instead of being silently ignored.
func LoadRules(path string) (domain.Rules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Rules{}, fmt.Errorf("cannot read rules file: %w", err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var rules domain.Rules
	if err := dec.Decode(&rules); err != nil {
		return domain.Rules{}, fmt.Errorf("invalid %s: %w", path, err)
	}
	return rules, nil
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
