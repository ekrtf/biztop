package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestObjectivesOverview(t *testing.T) {
	fecsDir := writeFEC(t, "Date;Libelle;Compte;Libelle du compte;Debit;Credit\n"+
		"01/01/2025;Invoice;706000;Prestations;0;1000\n"+
		"02/01/2025;Expense;613600;Software;125,50;25\n"+
		"03/01/2025;Tax;695000;Tax;999;0\n")
	rulesPath := writeRules(t, `management_fees: {}
objectives:
  - year: 2026
    revenue: 650000
    margin: 25
attio_types:
  - name: Projects
    billing: one-shot
    description: custom projects
`)
	cachePath := filepath.Join(t.TempDir(), "estimate.json")
	if err := os.WriteFile(cachePath, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := (Objectives{FecsDir: fecsDir, CachePath: cachePath, RulesPath: rulesPath}).Overview()
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Objectives) != 1 || len(out.Types) != 1 || string(out.Estimate) != `{"ok":true}` {
		t.Fatalf("unexpected overview metadata: %+v", out)
	}
	if o := out.Objectives[0]; o.Year != 2026 || o.Revenue != 650000 || o.Margin != 25 {
		t.Fatalf("unexpected objective: %+v", o)
	}
	actual := out.Actuals["2025"]
	if actual == nil || actual.CA != 1000 || actual.Charges != 100.5 || actual.Resultat != 899.5 {
		t.Fatalf("unexpected actuals: %+v", out.Actuals)
	}
}

func TestObjectivesOverviewReturnsLoadErrors(t *testing.T) {
	rulesPath := writeRules(t, "management_fees: {}\nattio_types: []\n")
	if _, err := (Objectives{FecsDir: filepath.Join(t.TempDir(), "missing"), RulesPath: rulesPath}).Overview(); err == nil {
		t.Fatal("Overview() error = nil, want missing FEC directory error")
	}
	fecsDir := writeFEC(t, "Date;Libelle;Compte;Libelle du compte;Debit;Credit\n")
	if _, err := (Objectives{FecsDir: fecsDir, RulesPath: filepath.Join(t.TempDir(), "missing.yml")}).Overview(); err == nil {
		t.Fatal("Overview() error = nil, want missing rules error")
	}
}

func TestObjectivesRefresh(t *testing.T) {
	dir := t.TempDir()
	installFakeCodexForService(t, dir, `{"deals":[],"by_type":{"Projects":42}}`)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	cachePath := filepath.Join(t.TempDir(), "estimate.json")
	rulesPath := writeRules(t, `management_fees: {}
attio_types:
  - name: Projects
    billing: one-shot
    description: custom projects
`)

	raw, err := (Objectives{RulesPath: rulesPath, CachePath: cachePath}).Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var estimate map[string]any
	if err := json.Unmarshal(raw, &estimate); err != nil {
		t.Fatal(err)
	}
	if estimate["fetched_at"] == "" || estimate["deals"] == nil {
		t.Fatalf("unexpected refreshed estimate: %+v", estimate)
	}
	if saved, err := os.ReadFile(cachePath); err != nil || !strings.Contains(string(saved), "fetched_at") {
		t.Fatalf("saved estimate = %q, err = %v", saved, err)
	}
}

func TestObjectivesRefreshReturnsErrors(t *testing.T) {
	if _, err := (Objectives{RulesPath: filepath.Join(t.TempDir(), "missing.yml")}).Refresh(context.Background()); err == nil {
		t.Fatal("Refresh() error = nil, want missing rules error")
	}
	rulesPath := writeRules(t, "management_fees: {}\nattio_types: []\n")
	if _, err := (Objectives{RulesPath: rulesPath}).Refresh(context.Background()); err == nil {
		t.Fatal("Refresh() error = nil, want missing attio types error")
	}
}

func installFakeCodexForService(t *testing.T, dir string, output string) {
	t.Helper()
	script := `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-o" ]; then
    shift
    out="$1"
  fi
  shift
done
printf '%s\n' '` + output + `' > "$out"
`
	if err := os.WriteFile(filepath.Join(dir, "codex"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}
