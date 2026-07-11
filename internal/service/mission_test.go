package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func missionFixture(t *testing.T, estimate string) Mission {
	t.Helper()
	// ACME is invoiced then paid by reference; BETA is paid by exact amount
	// through GoCardless (fees netted); GAMMA stays unpaid; KNAVE pays on
	// the spot with no receivable. OLD is a prior-year invoice.
	fecsDir := writeFEC(t, "Date;Libelle;Compte;Libelle du compte;Debit;Credit\n"+
		"15/01/2026;Facturation 202601-1 - ACME;706000;Prestations;0;10000\n"+
		"15/01/2026;Facturation 202601-1 - ACME;411000;Clients;12000;0\n"+
		"20/02/2026;2026011 virement - ACME;411000;Clients;0;12000\n"+
		"20/02/2026;2026011 virement - ACME;512001;Banque 1;12000;0\n"+
		"15/03/2026;Facturation 202603-2 - BETA;706000;Prestations;0;20000\n"+
		"15/03/2026;Facturation 202603-2 - BETA;411000;Clients;24000;0\n"+
		"10/04/2026;DAVAI-XYZ - GOCARDLESS SAS;411000;Clients;0;24000\n"+
		"10/04/2026;DAVAI-XYZ - GOCARDLESS SAS;512001;Banque 1;23990;0\n"+
		"10/04/2026;DAVAI-XYZ - GOCARDLESS SAS;627000;Services bancaires;10;0\n"+
		"10/05/2026;Facturation 202605-3 - GAMMA;706000;Prestations;0;5000\n"+
		"10/05/2026;Facturation 202605-3 - GAMMA;411000;Clients;6000;0\n"+
		"05/06/2026;Abo direct - KNAVE;706000;Prestations;0;500\n"+
		"05/06/2026;Abo direct - KNAVE;512001;Banque 1;500;0\n"+
		"20/03/2026;Loyer;613200;Loyers;5000;0\n"+
		"15/06/2025;Facturation 202506-9 - OLD;706000;Prestations;0;40000\n"+
		"15/06/2025;Facturation 202506-9 - OLD;411000;Clients;48000;0\n")
	rulesPath := writeRules(t, `management_fees: {}
objectives:
  - year: 2026
    revenue: 650000
    margin: 25
  - year: 2027
    revenue: 1200000
    margin: 28
attio_types:
  - name: Projects
    billing: one-shot
    description: custom projects
  - name: Maintenance
    billing: mrr
    description: recurring contracts
`)
	cachePath := filepath.Join(t.TempDir(), "estimate.json")
	if estimate != "" {
		if err := os.WriteFile(cachePath, []byte(estimate), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Mission{
		FecsDir:   fecsDir,
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

	// 2026 funnel: objectif 650k, pipeline 5000+600, CA 35500, cash
	// 10000 (ref match) + 20000 (amount match) + 500 (paid on the spot).
	r := out.Rows[0]
	if r.Year != 2026 || r.Objective != 650000 || r.Margin != 25 || r.CA != 35500 || r.Cash != 30500 {
		t.Fatalf("unexpected 2026 levels: %+v", r)
	}
	if r.Pipeline != 5600 || r.Projection != 41100 {
		t.Fatalf("2026 pipeline = %v, projection = %v, want 5600 and 41100", r.Pipeline, r.Projection)
	}
	if r.ResteVendre != 608900 || r.ResteFacturer != 614500 || r.ResteEncaisser != 5000 {
		t.Fatalf("unexpected 2026 restes: %+v", r)
	}
	if out.RunRate != 101483.33 { // 608900 / 6 months
		t.Fatalf("RunRate = %v, want 101483.33", out.RunRate)
	}

	next := out.Rows[1]
	if next.Year != 2027 || next.Pipeline != 2000 || next.CA != 0 || next.Cash != 0 || next.ResteVendre != 1198000 {
		t.Fatalf("unexpected 2027 row: %+v", next)
	}

	if out.MonthlyCA[0] != 10000 || out.MonthlyCA[2] != 20000 || out.MonthlyCA[4] != 5000 || out.MonthlyCA[5] != 500 {
		t.Fatalf("unexpected MonthlyCA: %v", out.MonthlyCA)
	}
	// Cash lands on the settlement month: ACME in February, BETA in April,
	// KNAVE on the spot in June.
	if out.MonthlyCash[1] != 10000 || out.MonthlyCash[3] != 20000 || out.MonthlyCash[5] != 500 || out.MonthlyCash[0] != 0 {
		t.Fatalf("unexpected MonthlyCash: %v", out.MonthlyCash)
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
	if r.Pipeline != 0 || r.ResteVendre != r.ResteFacturer || r.ResteVendre != 614500 {
		t.Fatalf("unexpected row without estimate: %+v", r)
	}
	if out.RunRate != 102416.67 { // 614500 / 6 months
		t.Fatalf("RunRate = %v, want 102416.67", out.RunRate)
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
