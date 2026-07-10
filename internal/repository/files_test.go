package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRules(t *testing.T) {
	rules, err := LoadRules(filepath.Join("..", "..", "rules.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rules.ManagementFees.Comptes) == 0 || len(rules.AttioTypes) == 0 {
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
