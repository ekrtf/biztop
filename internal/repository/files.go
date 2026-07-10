package repository

import (
	"encoding/json"
	"os"

	"biztop/internal/domain"
)

var defaultFeesConfig = domain.FeesConfig{
	LibellePatterns: []domain.PatternRule{
		{Pattern: "airbnb", Ratio: 1},
		{Pattern: "quote[ -]?part", Ratio: 1},
	},
	Comptes: []domain.CompteRule{
		{Compte: "625700", Ratio: 0.7}, // Restaurants et repas d'affaires
		{Compte: "625100", Ratio: 0.8}, // Frais de deplacements
	},
}

// LoadFeesConfig reads the management fees rules, creating the file with
// defaults on first use so it can be edited.
func LoadFeesConfig(path string) (domain.FeesConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		out, _ := json.MarshalIndent(defaultFeesConfig, "", "  ")
		if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
			return domain.FeesConfig{}, err
		}
		return defaultFeesConfig, nil
	}
	if err != nil {
		return domain.FeesConfig{}, err
	}
	var cfg domain.FeesConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return domain.FeesConfig{}, err
	}
	return cfg, nil
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
