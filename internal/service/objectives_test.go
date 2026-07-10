package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseObjectives(t *testing.T) {
	doc := "intro\nYearRevenue TargetNet Profit MarginApprox. Net ProfitTeam Size" +
		"2026650 – 800k€25–30%160 – 240k€6–7" +
		"20271.2 – 1.5M€28–33%340 – 500k€7–8" +
		"20282.0 – 2.5M€30–35%600 – 875k€8–9" +
		"20293.0 – 4.0M€30–35%900k – 1.4M€9–10" +
		"20304.5 – 6.0M€+32–35%1.4M – 2.1M€+10 max\n"

	objectives := ParseObjectives(doc)
	if len(objectives) != 5 {
		t.Fatalf("len(objectives) = %d, want 5: %+v", len(objectives), objectives)
	}
	if objectives[0].Year != 2026 || objectives[0].RevenueMin != 650000 || objectives[0].RevenueMax != 800000 || objectives[0].MarginMin != 25 || objectives[0].MarginMax != 30 || objectives[0].ProfitMin != 160000 || objectives[0].ProfitMax != 240000 || objectives[0].Team != "6–7" {
		t.Fatalf("unexpected first objective: %+v", objectives[0])
	}
	if objectives[1].Year != 2027 || objectives[1].RevenueMin != 1200000 || objectives[1].RevenueMax != 1500000 || objectives[1].Team != "7–8" {
		t.Fatalf("unexpected second objective: %+v", objectives[1])
	}
	if objectives[4].Year != 2030 || objectives[4].RevenueMin != 4500000 || objectives[4].RevenueMax != 6000000 || objectives[4].ProfitMin != 1400000 || objectives[4].ProfitMax != 2100000 || objectives[4].Team != "10 max" {
		t.Fatalf("unexpected last objective: %+v", objectives[4])
	}
}

func TestParseObjectivesReturnsNilWhenMissingTable(t *testing.T) {
	if got := ParseObjectives("no table"); got != nil {
		t.Fatalf("ParseObjectives(no table) = %+v, want nil", got)
	}
	if got := ParseObjectives("Revenue Target but no year"); got != nil {
		t.Fatalf("ParseObjectives(no year) = %+v, want nil", got)
	}
}

func TestToEuros(t *testing.T) {
	tests := []struct {
		num          string
		unit         string
		fallbackUnit string
		want         float64
	}{
		{num: "12", want: 12},
		{num: "12", unit: "k", want: 12000},
		{num: "1.5", unit: "M", want: 1500000},
		{num: "750", fallbackUnit: "k", want: 750000},
	}
	for _, tt := range tests {
		if got := toEuros(tt.num, tt.unit, tt.fallbackUnit); got != tt.want {
			t.Fatalf("toEuros(%q, %q, %q) = %v, want %v", tt.num, tt.unit, tt.fallbackUnit, got, tt.want)
		}
	}
}

func TestObjectivesOverview(t *testing.T) {
	fecsDir := writeFEC(t, "Date;Libelle;Compte;Libelle du compte;Debit;Credit\n"+
		"01/01/2025;Invoice;706000;Prestations;0;1000\n"+
		"02/01/2025;Expense;613600;Software;125,50;25\n"+
		"03/01/2025;Tax;695000;Tax;999;0\n")
	rulesPath := writeRules(t, `management_fees: {}
attio_types:
  - name: Projects
    billing: one-shot
    description: custom projects
`)
	docPath := filepath.Join(t.TempDir(), "objectives.md")
	if err := os.WriteFile(docPath, []byte("YearRevenue TargetNet Profit MarginApprox. Net ProfitTeam Size2026650 – 800k€25–30%160 – 240k€6–7"), 0o644); err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(t.TempDir(), "estimate.json")
	if err := os.WriteFile(cachePath, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := (Objectives{FecsDir: fecsDir, DocPath: docPath, CachePath: cachePath, RulesPath: rulesPath}).Overview()
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Objectives) != 1 || len(out.Types) != 1 || string(out.Estimate) != `{"ok":true}` {
		t.Fatalf("unexpected overview metadata: %+v", out)
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
