package service

import (
	"testing"
	"time"

	"biztop/internal/domain"
)

func cashEntry(t *testing.T, date, libelle, compte string, debit, credit float64) domain.Entry {
	t.Helper()
	d, err := time.Parse("02/01/2006", date)
	if err != nil {
		t.Fatal(err)
	}
	return domain.Entry{
		Date: d, Year: d.Year(), Month: int(d.Month()),
		Libelle: libelle, Compte: compte, Debit: debit, Credit: credit,
	}
}

func TestReconcileCashPartialAndCrossYear(t *testing.T) {
	e := func(date, libelle, compte string, debit, credit float64) domain.Entry {
		return cashEntry(t, date, libelle, compte, debit, credit)
	}
	entries := []domain.Entry{
		// 2025 invoice settled in 2026: cash on the invoice year, not the
		// settlement year, and excluded from the 2026 monthly chart.
		e("10/12/2025", "Facturation 202512-1 - OLD", "706000", 0, 1000),
		e("10/12/2025", "Facturation 202512-1 - OLD", "411000", 1200, 0),
		e("05/01/2026", "2025121 reglement - OLD", "411000", 0, 1200),
		e("05/01/2026", "2025121 reglement - OLD", "512001", 1200, 0),
		// Half-paid 2026 invoice: 50% of the HT is collected.
		e("10/02/2026", "Facturation 202602-2 - HALF", "706000", 0, 2000),
		e("10/02/2026", "Facturation 202602-2 - HALF", "411000", 2400, 0),
		e("15/03/2026", "acompte 202602-2 - HALF", "411000", 0, 1200),
		e("15/03/2026", "acompte 202602-2 - HALF", "512001", 1200, 0),
		// Avoir: no bank line, skipped by the matcher (nets the CA instead).
		e("20/03/2026", "Avoir 202603-9 - HALF", "706000", 500, 0),
		e("20/03/2026", "Avoir 202603-9 - HALF", "411000", 0, 600),
		// Settlement with no reference and no matching amount: dropped.
		e("25/03/2026", "mystere - GOCARDLESS SAS", "411000", 0, 777),
		e("25/03/2026", "mystere - GOCARDLESS SAS", "512001", 777, 0),
	}
	cash, monthly := reconcileCash(entries, 2026)
	if cash[2025] != 1000 {
		t.Fatalf("cash[2025] = %v, want 1000", cash[2025])
	}
	if cash[2026] != 1000 { // 2000 HT * 1200/2400 settled
		t.Fatalf("cash[2026] = %v, want 1000", cash[2026])
	}
	if monthly[0] != 0 { // the January settlement pays a 2025 invoice
		t.Fatalf("monthly[0] = %v, want 0", monthly[0])
	}
	if monthly[2] != 1000 {
		t.Fatalf("monthly[2] = %v, want 1000", monthly[2])
	}
}

func TestReconcileCashAmountMatchesOldestInvoice(t *testing.T) {
	e := func(date, libelle, compte string, debit, credit float64) domain.Entry {
		return cashEntry(t, date, libelle, compte, debit, credit)
	}
	entries := []domain.Entry{
		e("10/01/2026", "Facturation 202601-1 - A", "706000", 0, 1000),
		e("10/01/2026", "Facturation 202601-1 - A", "411000", 1200, 0),
		e("10/02/2026", "Facturation 202602-2 - B", "706000", 0, 1000),
		e("10/02/2026", "Facturation 202602-2 - B", "411000", 1200, 0),
		e("15/03/2026", "DAVAI-XYZ - GOCARDLESS SAS", "411000", 0, 1200),
		e("15/03/2026", "DAVAI-XYZ - GOCARDLESS SAS", "512001", 1195, 0),
	}
	cash, monthly := reconcileCash(entries, 2026)
	if cash[2026] != 1000 {
		t.Fatalf("cash[2026] = %v, want 1000 (one invoice paid)", cash[2026])
	}
	if monthly[2] != 1000 {
		t.Fatalf("monthly[2] = %v, want 1000", monthly[2])
	}
}

func TestInvoiceRef(t *testing.T) {
	tests := []struct {
		libelle string
		want    string
	}{
		{libelle: "Facturation 202601-133 - CHAUVIN PARIS", want: "202601133"},
		{libelle: "Facturation 202601-1 - STUDIO 54", want: "2026011"}, // client digits dropped
		{libelle: "Facturation 1 - ACME", want: ""},                    // too short
		{libelle: "no digits here"},
	}
	for _, tt := range tests {
		if got := invoiceRef(tt.libelle); got != tt.want {
			t.Fatalf("invoiceRef(%q) = %q, want %q", tt.libelle, got, tt.want)
		}
	}
}
