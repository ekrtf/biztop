package service

import (
	"os"
	"path/filepath"
	"testing"

	"biztop/internal/domain"
)

func TestYearsAndResolveYear(t *testing.T) {
	entries := []domain.Entry{{Year: 2026}, {Year: 2024}, {Year: 2026}, {Year: 2025}}
	years := Years(entries)
	if got, want := years, []int{2024, 2025, 2026}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("Years() = %v, want %v", got, want)
	}
	if got := ResolveYear(years, 0); got != 2026 {
		t.Fatalf("ResolveYear(latest) = %d, want 2026", got)
	}
	if got := ResolveYear(years, 2025); got != 2025 {
		t.Fatalf("ResolveYear(explicit) = %d, want 2025", got)
	}
	if got := ResolveYear(nil, 0); got != 0 {
		t.Fatalf("ResolveYear(empty) = %d, want 0", got)
	}
}

func TestComptaDataAggregatesYear(t *testing.T) {
	dir := writeFEC(t, "Date;Libelle;Compte;Libelle du compte;Debit;Credit\n"+
		"01/01/2024;Old invoice;706000;Prestations;0;10\n"+
		"01/01/2025;Invoice A;706000;Prestations;0;1000,005\n"+
		"02/01/2025;Invoice refund;706000;Prestations;50;0\n"+
		"03/02/2025;Sale;707000;Ventes;0;200\n"+
		"04/02/2025;Software;613600;Logiciels;100,004;10\n"+
		"05/03/2025;Restaurant;625700;Restaurants;80;0\n"+
		"06/03/2025;Tax;695000;Impot;999;0\n")

	out, err := (Compta{FecsDir: dir}).Data(0)
	if err != nil {
		t.Fatal(err)
	}
	if out.Year != 2025 {
		t.Fatalf("Year = %d, want 2025", out.Year)
	}
	if out.Totals.CA != 1150.01 || out.Totals.Charges != 170 || out.Totals.Resultat != 980.01 {
		t.Fatalf("unexpected totals: %+v", out.Totals)
	}
	if out.Monthly.CA[0] != 950.01 || out.Monthly.CA[1] != 200 || out.Monthly.Charges[1] != 90 || out.Monthly.Charges[2] != 80 {
		t.Fatalf("unexpected monthly values: %+v", out.Monthly)
	}
	if len(out.Revenue) != 2 || out.Revenue[0].Compte != "706000" || out.Revenue[0].Total != 950.01 {
		t.Fatalf("unexpected revenue rows: %+v", out.Revenue)
	}
	if len(out.Charges) != 2 || out.Charges[0].Compte != "613600" || out.Charges[0].Total != 90 {
		t.Fatalf("unexpected charge rows: %+v", out.Charges)
	}
}

func TestComptaDataReturnsRepositoryError(t *testing.T) {
	if _, err := (Compta{FecsDir: filepath.Join(t.TempDir(), "missing")}).Data(2025); err == nil {
		t.Fatal("Data() error = nil, want missing FEC directory error")
	}
}

func TestTransactionsFiltersSortsAndTotals(t *testing.T) {
	dir := writeFEC(t, "Date;Libelle;Compte;Libelle du compte;Debit;Credit\n"+
		"10/02/2025;Later;613600;Logiciels;40;0\n"+
		"01/02/2025;Earlier;613600;Logiciels;10;1\n"+
		"01/03/2025;Other month;613600;Logiciels;99;0\n"+
		"01/02/2025;Other account;625700;Restaurants;50;0\n")

	out, err := (Compta{FecsDir: dir}).Transactions(2025, "613600", 2)
	if err != nil {
		t.Fatal(err)
	}
	if out.Libelle != "Logiciels" || out.TotalDebit != 50 || out.TotalCredit != 1 {
		t.Fatalf("unexpected transaction list: %+v", out)
	}
	if len(out.Transactions) != 2 || out.Transactions[0].Libelle != "Earlier" || out.Transactions[1].Libelle != "Later" {
		t.Fatalf("transactions not filtered and sorted: %+v", out.Transactions)
	}
}

func writeFEC(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fec.csv"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
