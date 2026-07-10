package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFeesComputeMatchesRulesAndAdjustsResult(t *testing.T) {
	fecsDir := writeFEC(t, "Date;Libelle;Compte;Libelle du compte;Debit;Credit\n"+
		"01/01/2025;Invoice;706000;Prestations;0;1000\n"+
		"02/01/2025;Client lunch;625700;Restaurants;100;0\n"+
		"03/01/2025;Train;625100;Travel;100;0\n"+
		"04/02/2025;Airbnb stay;613600;Logement;50;0\n"+
		"05/02/2025;Deliveroo dinner;625700;Restaurants;100;0\n"+
		"06/03/2024;Old lunch;625700;Restaurants;100;0\n")
	rulesPath := writeRules(t, `management_fees:
  comptes:
    - compte: "625700"
      ratio: 0.7
    - compte: "625100"
      ratio: 0.8
  libelle_patterns: [airbnb]
  exclude_patterns: [deliveroo]
attio_types:
  - name: Projects
    billing: one-shot
    description: custom projects
`)

	out, err := (Fees{FecsDir: fecsDir, RulesPath: rulesPath}).Compute(0)
	if err != nil {
		t.Fatal(err)
	}
	if out.Year != 2025 || out.Total != 200 || out.Monthly[0] != 150 || out.Monthly[1] != 50 {
		t.Fatalf("unexpected fee totals: %+v", out)
	}
	if out.Resultat != 650 || out.ResultatAjuste != 850 {
		t.Fatalf("unexpected adjusted result: resultat=%v adjusted=%v", out.Resultat, out.ResultatAjuste)
	}
	if len(out.Transactions) != 3 || out.Transactions[0].Fee != 70 || out.Transactions[1].Fee != 80 || out.Transactions[2].Fee != 50 {
		t.Fatalf("unexpected fee transactions: %+v", out.Transactions)
	}
}

func TestFeesComputeReportsInvalidRegex(t *testing.T) {
	fecsDir := writeFEC(t, "Date;Libelle;Compte;Libelle du compte;Debit;Credit\n01/01/2025;Invoice;706000;Prestations;0;1000\n")
	rulesPath := writeRules(t, `management_fees:
  libelle_patterns: ["["]
attio_types: []
`)

	_, err := (Fees{FecsDir: fecsDir, RulesPath: rulesPath}).Compute(2025)
	if err == nil || !strings.Contains(err.Error(), "invalid pattern") {
		t.Fatalf("Compute() error = %v, want invalid pattern", err)
	}
}

func TestFeesComputeReturnsLoadErrors(t *testing.T) {
	rulesPath := writeRules(t, "management_fees: {}\nattio_types: []\n")
	if _, err := (Fees{FecsDir: filepath.Join(t.TempDir(), "missing"), RulesPath: rulesPath}).Compute(2025); err == nil {
		t.Fatal("Compute() error = nil, want missing FEC directory error")
	}
	fecsDir := writeFEC(t, "Date;Libelle;Compte;Libelle du compte;Debit;Credit\n")
	if _, err := (Fees{FecsDir: fecsDir, RulesPath: filepath.Join(t.TempDir(), "missing.yml")}).Compute(2025); err == nil {
		t.Fatal("Compute() error = nil, want missing rules error")
	}
}

func writeRules(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rules.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
