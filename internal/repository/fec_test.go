package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEntries(t *testing.T) {
	dir := t.TempDir()
	fec := "\ufeffDate;Libelle;Compte;Libelle du compte;Debit;Credit\n" +
		"01/02/2026;Invoice;706000;Prestations;0,00;1234,56\n" +
		"bad date;Ignored;706000;Prestations;0;99\n" +
		"03/02/2026;Refund;613600;Software;10.50;2,25\n"
	if err := os.WriteFile(filepath.Join(dir, "2026.csv"), []byte(fec), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignored"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := LoadEntries(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2: %+v", len(entries), entries)
	}
	if entries[0].Year != 2026 || entries[0].Month != 2 || entries[0].Credit != 1234.56 {
		t.Fatalf("first entry parsed incorrectly: %+v", entries[0])
	}
	if entries[1].Debit != 10.5 || entries[1].Credit != 2.25 {
		t.Fatalf("amounts parsed incorrectly: %+v", entries[1])
	}
}

func TestLoadEntriesSkipsEmptyCSV(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "empty.csv"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := LoadEntries(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("len(entries) = %d, want 0", len(entries))
	}
}

func TestLoadEntriesReturnsCSVReadErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "bad.csv"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadEntries(dir); err == nil {
		t.Fatal("LoadEntries() error = nil, want CSV read error")
	}
}

func TestLoadEntriesReturnsReadErrors(t *testing.T) {
	if _, err := LoadEntries(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("LoadEntries() error = nil, want missing directory error")
	}
}

func TestParseAmount(t *testing.T) {
	tests := map[string]float64{
		"":       0,
		"  ":     0,
		"12,34":  12.34,
		"12.34":  12.34,
		"n/a":    0,
		" 5,00 ": 5,
	}
	for input, want := range tests {
		if got := parseAmount(input); got != want {
			t.Fatalf("parseAmount(%q) = %v, want %v", input, got, want)
		}
	}
}
