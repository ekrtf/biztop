package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func missionFixture(t *testing.T, estimate string) Mission {
	t.Helper()
	fecsDir := writeFEC(t, "Date;Libelle;Compte;Libelle du compte;Debit;Credit\n"+
		"15/01/2026;Facturation 1 - ACME;706000;Prestations;0;10000\n"+
		"15/03/2026;Facturation 2 - ACME;706000;Prestations;0;20000\n"+
		"20/03/2026;Loyer;613200;Loyers;5000;0\n"+
		"15/06/2025;Facturation 3 - ACME;706000;Prestations;0;40000\n")
	rulesPath := writeRules(t, `management_fees: {}
attio_types:
  - name: Projects
    billing: one-shot
    description: custom projects
  - name: Maintenance
    billing: mrr
    description: recurring contracts
`)
	docPath := filepath.Join(t.TempDir(), "objectives.md")
	if err := os.WriteFile(docPath, []byte("YearRevenue TargetNet Profit MarginApprox. Net ProfitTeam Size"+
		"2026650 – 800k€25–30%160 – 240k€6–7"+
		"20271.2 – 1.5M€28–33%340 – 500k€7–8"), 0o644); err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(t.TempDir(), "estimate.json")
	if estimate != "" {
		if err := os.WriteFile(cachePath, []byte(estimate), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Mission{
		FecsDir:   fecsDir,
		DocPath:   docPath,
		CachePath: cachePath,
		RulesPath: rulesPath,
		Now:       func() time.Time { return time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC) },
	}
}

func TestMissionCompute(t *testing.T) {
	// One-shot deal counted once, MRR deal projected from July to December
	// (6 months), both weighted by probability.
	out, err := missionFixture(t, `{
		"deals": [
			{"name": "Big project", "type": "Projects", "amount_eur": 10000, "probability": 0.5, "expected_month": "2026-08"},
			{"name": "Hosting", "type": "Maintenance", "amount_eur": 100, "probability": 1, "expected_month": "2026-07"},
			{"name": "Next year", "type": "Projects", "amount_eur": 8000, "probability": 0.25, "expected_month": "2027-02"},
			{"name": "Broken", "type": "Projects", "amount_eur": 1000, "probability": 1, "expected_month": "n/a"}
		],
		"fetched_at": "2026-07-10T09:00:00+02:00"
	}`).Compute()
	if err != nil {
		t.Fatal(err)
	}
	if out.Year != 2026 || out.Month != 7 || out.MonthsLeft != 6 {
		t.Fatalf("unexpected clock: %+v", out)
	}
	if !out.HasEstimate || out.FetchedAt != "2026-07-10T09:00:00+02:00" {
		t.Fatalf("unexpected estimate metadata: %+v", out)
	}
	if len(out.Rows) != 2 {
		t.Fatalf("len(Rows) = %d, want 2: %+v", len(out.Rows), out.Rows)
	}

	r := out.Rows[0]
	if r.Year != 2026 || r.ObjectiveMin != 650000 || r.CA != 30000 || r.Resultat != 25000 {
		t.Fatalf("unexpected 2026 actuals: %+v", r)
	}
	if r.Pipeline != 5600 || r.Projection != 35600 {
		t.Fatalf("2026 pipeline = %v, projection = %v, want 5600 and 35600", r.Pipeline, r.Projection)
	}
	if r.ResteCompta != 620000 || r.Reste != 614400 || r.ResteResultat != 135000 {
		t.Fatalf("unexpected 2026 restes: %+v", r)
	}
	if out.RunRate != 102400 { // 614400 / 6 months
		t.Fatalf("RunRate = %v, want 102400", out.RunRate)
	}

	next := out.Rows[1]
	if next.Year != 2027 || next.Pipeline != 2000 || next.CA != 0 || next.Reste != 1198000 {
		t.Fatalf("unexpected 2027 row: %+v", next)
	}

	if out.MonthlyCA[0] != 10000 || out.MonthlyCA[2] != 20000 || out.MonthlyCA[5] != 0 {
		t.Fatalf("unexpected MonthlyCA: %v", out.MonthlyCA)
	}
	// July: 100 MRR; August: 100 MRR + 5000 one-shot; December: 100 MRR.
	if out.MonthlyPipeline[6] != 100 || out.MonthlyPipeline[7] != 5100 || out.MonthlyPipeline[11] != 100 {
		t.Fatalf("unexpected MonthlyPipeline: %v", out.MonthlyPipeline)
	}
}

func TestMissionComputeWithoutEstimate(t *testing.T) {
	out, err := missionFixture(t, "").Compute()
	if err != nil {
		t.Fatal(err)
	}
	if out.HasEstimate || out.FetchedAt != "" {
		t.Fatalf("expected no estimate: %+v", out)
	}
	r := out.Rows[0]
	if r.Pipeline != 0 || r.Reste != r.ResteCompta || r.Reste != 620000 {
		t.Fatalf("unexpected row without estimate: %+v", r)
	}
	if out.RunRate != 103333.33 { // 620000 / 6 months
		t.Fatalf("RunRate = %v, want 103333.33", out.RunRate)
	}
}

func TestMissionComputeReturnsLoadErrors(t *testing.T) {
	rulesPath := writeRules(t, "management_fees: {}\nattio_types: []\n")
	if _, err := (Mission{FecsDir: filepath.Join(t.TempDir(), "missing"), RulesPath: rulesPath}).Compute(); err == nil {
		t.Fatal("Compute() error = nil, want missing FEC directory error")
	}
	fecsDir := writeFEC(t, "Date;Libelle;Compte;Libelle du compte;Debit;Credit\n")
	if _, err := (Mission{FecsDir: fecsDir, RulesPath: filepath.Join(t.TempDir(), "missing.yml")}).Compute(); err == nil {
		t.Fatal("Compute() error = nil, want missing rules error")
	}
}

func TestParseMonth(t *testing.T) {
	tests := []struct {
		in          string
		year, month int
	}{
		{in: "2026-07", year: 2026, month: 7},
		{in: "2026-12-01", year: 2026, month: 12},
		{in: "n/a"},
		{in: ""},
		{in: "2026-13"},
		{in: "0000-05"},
	}
	for _, tt := range tests {
		if y, m := parseMonth(tt.in); y != tt.year || m != tt.month {
			t.Fatalf("parseMonth(%q) = %d, %d, want %d, %d", tt.in, y, m, tt.year, tt.month)
		}
	}
}
