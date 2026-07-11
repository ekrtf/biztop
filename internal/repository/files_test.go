package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRules(t *testing.T) {
	rules, err := LoadRules(filepath.Join("..", "..", "rules.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rules.ManagementFees.Comptes) == 0 || len(rules.AttioTypes) == 0 || len(rules.Objectives) == 0 {
		t.Fatalf("rules.yml incomplete: %+v", rules)
	}
}

func TestLoadRulesRejectsUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rules.yml")
	typo := "management_fees:\n  exclude: [deliveroo]\n"
	if err := os.WriteFile(path, []byte(typo), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRules(path); err == nil {
		t.Fatal("a mistyped key must be an error, not silently ignored")
	}
}

func TestLoadRulesRejectsInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rules.yml")
	if err := os.WriteFile(path, []byte("management_fees: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRules(path); err == nil {
		t.Fatal("LoadRules() error = nil, want invalid YAML error")
	}
}

func TestLoadRulesReturnsReadErrors(t *testing.T) {
	if _, err := LoadRules(filepath.Join(t.TempDir(), "missing.yml")); err == nil {
		t.Fatal("LoadRules() error = nil, want missing file error")
	}
}

func TestLoadAndSaveEstimate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "estimate.json")
	if got := LoadEstimate(path); got != nil {
		t.Fatalf("LoadEstimate(missing) = %s, want nil", got)
	}
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := LoadEstimate(path); got != nil {
		t.Fatalf("LoadEstimate(invalid) = %s, want nil", got)
	}

	raw, err := SaveEstimate(path, map[string]any{"ok": true})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(raw) {
		t.Fatalf("SaveEstimate() returned invalid JSON: %s", raw)
	}
	if got := LoadEstimate(path); string(got) != "{\n  \"ok\": true\n}\n" {
		t.Fatalf("LoadEstimate(saved) = %q", got)
	}
	if _, err := SaveEstimate(path, map[string]any{"bad": make(chan int)}); err == nil {
		t.Fatal("SaveEstimate(unmarshalable) error = nil, want marshal error")
	}
	if _, err := SaveEstimate(t.TempDir(), map[string]any{"ok": true}); err == nil {
		t.Fatal("SaveEstimate(directory) error = nil, want write error")
	}
}
